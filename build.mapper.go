package rdrr

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/maseology/goHydro/grid"
)

func (s *Structure) buildMapper(lufp, sgfp, gwfp string,
	iksat func(*grid.Definition, []int, []int) ([]float64, []int),
	xlu func(*grid.Definition, string, []int) SurfaceSet,
) Mapper {
	var wg sync.WaitGroup

	// data generally comes in raster format, here they're recast as arrays
	var ilu, isg, igw, icov []int
	var ksat, fimp, ifct []float64
	var fngwc []float64

	fileExists := func(path string) (int64, bool) {
		if fi, err := os.Stat(path); err == nil {
			return fi.Size(), true
		} else if os.IsNotExist(err) {
			return 0, false
		} else {
			// log.Fatalf("mmio.FileExists: %v", err)
			return 0, false
		}
	}

	readLU := func(gd *grid.Definition, lufp string) {
		defer wg.Done()
		if lufp == "" {
			return
		}
		tt := time.Now()

		// checkforfile := func(fp string) {
		// 	if _, ok := fileExists(fp); !ok {
		// 		log.Fatalf(" getMappings.readLU file not found: %s", fp)
		// 	}
		// }

		// // load data
		// loadReal := func(fp string) []float64 {
		// 	checkforfile(fp)
		// 	fmt.Printf("   loading: %s\n", fp)
		// 	var g grid.Real
		// 	g.NewGD32(fp, gd)
		// 	m := make(map[int]float64, len(g.A))
		// 	for k, v := range g.A {
		// 		if v < 0. {
		// 			m[k] = 0.
		// 		} else {
		// 			m[k] = v
		// 		}
		// 	}
		// 	aout := make([]float64, s.Nc)
		// 	for i, c := range s.Cids {
		// 		if v, ok := m[c]; ok {
		// 			aout[i] = v
		// 		} else {
		// 			panic("getMappings.loadReal error: " + fp)
		// 		}
		// 	}
		// 	return aout
		// }
		// // load indices
		// loadIndx := func(fp string) ([]int, []int) {
		// 	checkforfile(fp)
		// 	fmt.Printf("   loading: %s\n", fp)
		// 	var g grid.Indx
		// 	g.LoadGDef(gd)
		// 	g.NewShort(fp, true)
		// 	m := g.Values()
		// 	aout := make([]int, s.Nc)
		// 	for i, c := range s.Cids {
		// 		if v, ok := m[c]; ok {
		// 			aout[i] = v
		// 		} else {
		// 			panic("getMappings.loadIndx error: " + fp)
		// 		}
		// 	}
		// 	return aout, g.UniqueValues()
		// }
		// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
		var ulu []int
		// if _, ok := fileExists(lufp + "-surfaceid.bil"); ok {
		// 	ilu, ulu = loadIndx(lufp + "-surfaceid.bil")
		// 	icov, _ = loadIndx(lufp + "-canopyid.bil")
		// 	fimp = loadReal(lufp + "-perimp.bil")
		// 	ifct = loadReal(lufp + "-percov.bil") // fraction cover (to be adjusted below)
		// } else {
		ss := xlu(gd, lufp, s.Cids)
		ilu, ulu, icov, fimp, ifct = ss.Ilu, ss.Ulu, ss.Icov, ss.Fimp, ss.Ifct
		// }

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

		fmt.Printf(" %s - %v\n", "LU loaded", time.Since(tt))
	}

	readSG := func(gd *grid.Definition, sgfp string) {
		defer wg.Done()
		tt := time.Now()

		// load index
		loadIndx := func(fp string) ([]int, []int) {
			if _, ok := fileExists(fp); !ok {
				log.Fatalf(" getMappings.readSG.loadIndx file not found: %s", fp)
			}
			fmt.Printf("   loading: %s\n", fp)
			var g grid.Indx
			switch filepath.Ext(fp) {
			case ".bil":
				g.LoadGDef(gd)
				g.NewShort(fp, true)
			case ".indx":
				if _, b := fileExists(fp + ".gdef"); !b {
					g.LoadGDef(gd)
				}
				g.New(fp, true)
			default:
				log.Fatalf("unrecognized file format: " + fp)
			}
			aout := make([]int, s.Nc)
			for i, c := range s.Cids {
				if v, ok := g.A[c]; ok {
					aout[i] = v
				} else {
					panic("getMappings.readSG.loadIndx error: " + fp)
				}
			}
			return aout, g.UniqueValues()
		}
		isg, _ = loadIndx(sgfp)
		ksat, _ = iksat(gd, s.Cids, isg) // same size and order of usg[]
		fmt.Printf(" %s - %v\n", "SG loaded", time.Since(tt))
	}

	readGW := func(gd *grid.Definition, gwfp string) {
		defer wg.Done()
		tt := time.Now()

		if gwfp == "" { // all 1 GW zone
			fngwc, igw = s.buildGWzone(nil)
		} else {
			// load index
			loadIndx := func(fp string) (map[int]int, []int) {
				if _, ok := fileExists(fp); !ok {
					log.Fatalf(" getMappings.readGW file not found: %s", fp)
				}
				fmt.Printf("   loading: %s\n", fp)
				var g grid.Indx
				switch filepath.Ext(fp) {
				case ".bil":
					g.LoadGDef(gd)
					g.NewShort(fp, true)
				case ".indx":
					if _, b := fileExists(fp + ".gdef"); !b {
						g.LoadGDef(gd)
					}
					g.New(fp, true)
				default:
					log.Fatalf("unrecognized file format: " + fp)
				}
				return g.A, g.UniqueValues()
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

		fmt.Printf(" %s - %v\n", "GW zones loaded", time.Since(tt))
	}

	wg.Add(3)
	go readLU(s.GD, lufp)
	go readSG(s.GD, sgfp)
	go readGW(s.GD, gwfp)
	wg.Wait()

	mx := make(map[int]int, s.Nc) // grid cell id to array index cross-reference
	for i, c := range s.Cids {
		mx[c] = i
	}

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
