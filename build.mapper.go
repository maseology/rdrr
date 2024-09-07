package rdrr

import (
	"fmt"
	"os"
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
	var ksat, fimp, fint []float64
	var fngwc []float64

	fileExists := func(path string) (int64, bool) {
		if fi, err := os.Stat(path); err == nil {
			return fi.Size(), true
		} else if os.IsNotExist(err) {
			return 0, false
		} else {
			return 0, false
		}
	}

	readLU := func(gd *grid.Definition, lufp string) {
		defer wg.Done()
		if lufp == "" {
			return
		}
		tt := time.Now()
		var ulu []int
		ss := xlu(gd, lufp, s.Cids)
		ilu, ulu, icov, fimp, fint = ss.Ilu, ss.Ulu, ss.Icov, ss.Fimp, ss.Fint

		// adjust cover (convert to fint; note ss.Fint originally held canopy cover fraction)
		for i := range s.Cids {
			fint[i] *= relativeCover(icov[i], ilu[i])
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

		fmt.Printf(" > %s - %v\n", "LU loaded", time.Since(tt))
	}

	readSG := func(gd *grid.Definition, sgfp string) {
		defer wg.Done()
		tt := time.Now()

		// load index
		loadIndx := func(fp string) ([]int, []int) {
			if _, ok := fileExists(fp); !ok {
				panic(fmt.Sprintf("getMappings.readSG.loadIndx file not found: %s", fp))
			}
			fmt.Printf("   loading: %s\n", fp)
			var g grid.Indx
			g.GD = gd
			g.New(fp)

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
		isgt, _ := loadIndx(sgfp)
		ksat, isg = iksat(gd, s.Cids, isgt) // same size and order of usg[]
		fmt.Printf(" > %s - %v\n", "SG loaded", time.Since(tt))
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
					panic(fmt.Sprintf("getMappings.readGW.loadIndx file not found: %s", fp))
				}
				fmt.Printf("   loading: %s\n", fp)
				var g grid.Indx
				g.GD = gd
				g.New(fp)
				return g.A, g.UniqueValues()
			}
			mgw, _ := loadIndx(gwfp)
			agw := make([]int, s.Nc)
			var nulls []int
			for i, c := range s.Cids {
				if gid, ok := mgw[c]; ok {
					if gid == 255 {
						nulls = append(nulls, c)
					}
					agw[i] = gid
				} else {
					panic("getMappings.readGW groundwater cellID error")
				}
			}

			if len(nulls) > 0 {
				panic(fmt.Sprintf("getMappings.readGW groundwater ID null value found at cells:\n %d", nulls))
			}

			fngwc, igw = s.buildGWzone(agw)
		}

		fmt.Printf(" > %s - %v\n", "GW zones loaded", time.Since(tt))
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
		Fint:  fint,
		Fngwc: fngwc,
	}
}
