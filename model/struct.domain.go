package model

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/mmio"
)

// Domain holds all data and is the parent to all sub models
type Domain struct {
	Frc     *FORC // forcing (variable) data
	Strc    *STRC // structural (unchanging) data (eg, topography, solar irradiation fractions)
	Mpr     *MAPR // land use/surficial geology mapping for parameter assignment
	Obs     *OBS  // model observations/calibration targets
	Nc, Ngw int   // number of cells; number of groundwater reservoirs
	// mons    []int // monitor cell IDs
	// Fgwnc   []float64 // cell count of each gw zone
	Dir string // model directory/prefix
}

// LoadDomain loads all data from which sub-domain scale models can be derived
func LoadDomain(mdlprfx string) *Domain {
	fmt.Println("Loading Master Domain..")

	rootdir := mmio.GetFileDir(mdlprfx)
	frc, strc, mpr, obs := func() (*FORC, *STRC, *MAPR, *OBS) {
		var wg sync.WaitGroup

		var frc *FORC
		var strc *STRC
		var mapr *MAPR
		var obs *OBS

		wg.Add(3)

		go func() {
			defer wg.Done()
			var err error
			if frc, err = LoadGobFORC(mdlprfx + "domain.FORC.gob"); err != nil {
				log.Fatalf("%v", err)
			}
		}()

		go func() {
			defer wg.Done()
			var err error
			if strc, err = LoadGobSTRC(mdlprfx + "domain.STRC.gob"); err != nil {
				log.Fatalf("%v", err)
			}
		}()

		go func() {
			defer wg.Done()
			var err error
			if mapr, err = LoadGobMAPR(mdlprfx + "domain.MAPR.gob"); err != nil {
				log.Fatalf("%v", err)
			}

		}()

		wg.Wait()

		// load model observations, calibration targets
		func() {
			m := make(map[int]int)
			for i, c := range strc.CIDs {
				m[c] = i
			}
			obs = collectOBS(frc, mdlprfx)
			if mmio.DirExists(rootdir + "/obs/") {
				obs.AddFluxCsv(rootdir+"/obs/", m, strc.Wcell*strc.Wcell)
			}
		}()

		return frc, strc, mapr, obs
	}()

	// ugw := func() []int {
	// 	u := make([]int, 0, len(mpr.Fngwc))
	// 	for k := range mpr.Fngwc {
	// 		u = append(u, k)
	// 	}
	// 	sort.Ints(u)
	// 	return u
	// }()
	// fgnc := make([]float64, len(ugw))
	// for i, k := range ugw {
	// 	fgnc[i] = mpr.GW[k].Fnc
	// }

	return &Domain{
		Frc:  frc,
		Strc: strc,
		Mpr:  mpr,
		Obs:  obs,
		Nc:   len(strc.CIDs),
		Ngw:  len(mpr.Fngwc),
		// mons: mons,
		// Fgwnc: fgnc,
		Dir: rootdir,
	}
}
