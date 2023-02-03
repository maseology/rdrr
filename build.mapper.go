package rdrr

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
)

func (s *Structure) buildMapper(lufp, sgfp, gwfp string,
	iksat func([]int) []float64,
	xlu func(*grid.Definition, string, map[int]int) SurfaceSet,
) Mapper {
	var wg sync.WaitGroup

	// data generally comes in raster format, here they're recast as arrays
	var ilu, isg, igw, icov []int
	var ksat, fimp, ifct []float64
	var fngwc []float64

	mx := make(map[int]int, s.Nc) // grid cell id to array index cross-reference
	for i, c := range s.Cids {
		mx[c] = i
	}

	readLU := func(gd *grid.Definition, lufp string) {
		defer wg.Done()
		if lufp == "" {
			return
		}
		tt := mmio.NewTimer()

		checkforfile := func(fp string) {
			if _, ok := mmio.FileExists(fp); !ok {
				log.Fatalf(" getMappings.readLU file not found: %s", fp)
			}
		}

		// load data
		loadReal := func(fp string) []float64 {
			checkforfile(fp)
			fmt.Printf(" loading: %s\n", fp)
			var g grid.Real
			g.NewGD32(fp, gd)
			m := make(map[int]float64, len(g.A))
			for k, v := range g.A {
				if v < 0. {
					m[k] = 0.
				} else {
					m[k] = v
				}
			}
			aout := make([]float64, s.Nc)
			for i, c := range s.Cids {
				if v, ok := m[c]; ok {
					aout[i] = v
				} else {
					panic("getMappings.loadReal error: " + fp)
				}
			}
			return aout
		}
		// load indices
		loadIndx := func(fp string) ([]int, []int) {
			checkforfile(fp)
			fmt.Printf(" loading: %s\n", fp)
			var g grid.Indx
			g.LoadGDef(gd)
			g.NewShort(fp, true)
			m := g.Values()
			aout := make([]int, s.Nc)
			for i, c := range s.Cids {
				if v, ok := m[c]; ok {
					aout[i] = v
				} else {
					panic("getMappings.loadIndx error: " + fp)
				}
			}
			return aout, g.UniqueValues()
		}

		var ulu []int
		if _, ok := mmio.FileExists(lufp + "-surfaceid.bil"); ok {
			ilu, ulu = loadIndx(lufp + "-surfaceid.bil")
			icov, _ = loadIndx(lufp + "-canopyid.bil")
			fimp = loadReal(lufp + "-perimp.bil")
			ifct = loadReal(lufp + "-percov.bil") // fraction cover (to be adjusted below)
		} else {
			ss := xlu(gd, lufp, mx)
			ilu, ulu, icov, fimp, ifct = ss.Ilu, ss.Ulu, ss.Icov, ss.Fimp, ss.Ifct
		}

		// adjust cover (convert to ifct)
		for i := range s.Cids {
			ifct[i] *= relativeCover(icov[i], ilu[i])
		}

		// force stream cells to Channel type
		strms, _ := s.buildStreams() // collect stream cells
		for _, c := range strms {
			ilu[c] = Channel
		}
		if func() bool { // check if channels already exist in ulu, if not, add
			for _, c := range ulu {
				if c == Channel {
					return false
				}
			}
			return true
		}() {
			ulu = append(ulu, Channel)
		}

		tt.Lap("LU loaded")
	}

	readSG := func(gd *grid.Definition, sgfp string) {
		defer wg.Done()
		tt := mmio.NewTimer()

		// load index
		loadIndx := func(fp string) ([]int, []int) {
			if _, ok := mmio.FileExists(fp); !ok {
				log.Fatalf(" getMappings.readSG.loadIndx file not found: %s", fp)
			}
			fmt.Printf(" loading: %s\n", fp)
			var g grid.Indx
			switch mmio.GetExtension(fp) {
			case ".bil":
				g.LoadGDef(gd)
				g.NewShort(fp, true)
			case ".indx":
				if _, b := mmio.FileExists(fp + ".gdef"); !b {
					g.LoadGDef(gd)
				}
				g.New(fp, true)
			default:
				log.Fatalf("unrecognized file format: " + fp)
			}
			m := g.Values()
			aout := make([]int, s.Nc)
			for i, c := range s.Cids {
				if v, ok := m[c]; ok {
					aout[i] = v
				} else {
					panic("getMappings.readSG.loadIndx error: " + fp)
				}
			}
			return aout, g.UniqueValues()
		}
		isg, _ = loadIndx(sgfp)
		ksat = iksat(isg) // same size and order of usg[]
		tt.Lap("SG loaded")
	}

	readGW := func(gd *grid.Definition, gwfp string) {
		defer wg.Done()
		tt := mmio.NewTimer()

		if gwfp == "" { // all 1 GW zone
			fngwc, igw = s.buildGWzone(nil)
		} else {
			// load index
			loadIndx := func(fp string) (map[int]int, []int) {
				if _, ok := mmio.FileExists(fp); !ok {
					log.Fatalf(" getMappings.readGW file not found: %s", fp)
				}
				fmt.Printf(" loading: %s\n", fp)
				var g grid.Indx
				switch mmio.GetExtension(fp) {
				case ".bil":
					g.LoadGDef(gd)
					g.NewShort(fp, true)
				case ".indx":
					if _, b := mmio.FileExists(fp + ".gdef"); !b {
						g.LoadGDef(gd)
					}
					g.New(fp, true)
				default:
					log.Fatalf("unrecognized file format: " + fp)
				}
				return g.Values(), g.UniqueValues()
			}
			mgw, _ := loadIndx(gwfp)
			agw := make([]int, s.Nc)
			for i, c := range s.Cids {
				if gid, ok := mgw[c]; ok {
					agw[i] = gid
				} else {
					panic("groundwater id error")
				}
			}
			fngwc, igw = s.buildGWzone(agw)
		}

		tt.Lap("GW zones loaded")
	}

	wg.Add(3)
	go readLU(s.GD, lufp)
	go readSG(s.GD, sgfp)
	go readGW(s.GD, gwfp)
	wg.Wait()

	return Mapper{
		Mx:    mx,
		Ilu:   ilu,
		Isg:   isg,
		Igw:   igw,
		Icov:  icov,
		Ksat:  ksat,
		Fimp:  fimp,
		Ifct:  ifct,
		Fngwc: fngwc,
	}
}
