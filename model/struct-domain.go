package model

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/mmio"
)

// Domain holds all data and is the parent to all sub models
type Domain struct {
	Frc  *FORC  // forcing (variable) data
	Strc *STRC  // structural (unchanging) data (eg, topography, solar irradiation fractions)
	rtr  *RTR   // subwatershed topology
	mpr  *MAPR  // land use/surficial geology mapping for parameter assignment
	mons []int  // monitor cell IDs
	dir  string // model directory/prefix
}

// LoadDomain loads all data from which sub-domain scale models can be derived
func LoadDomain(mdlprfx string) *Domain {
	fmt.Println("Loading Master Domain..")

	frc, strc, rtr, mpr, mons := func() (*FORC, *STRC, *RTR, *MAPR, []int) {
		var wg sync.WaitGroup
		wg.Add(5)
		var frc *FORC
		go func() {
			defer wg.Done()
			var err error
			if frc, err = LoadGobFORC(mdlprfx + "domain.FORC.gob"); err != nil {
				log.Fatalf("%v", err)
			}
			frc.mt = make([]int, len(frc.T))
			for k, dt := range frc.T {
				frc.mt[k] = int(dt.Month())
			}
		}()

		var strc *STRC
		go func() {
			defer wg.Done()
			var err error
			if strc, err = LoadGobSTRC(mdlprfx + "domain.STRC.gob"); err != nil {
				log.Fatalf("%v", err)
			}
		}()

		var rtr *RTR
		go func() {
			defer wg.Done()
			var err error
			if rtr, err = LoadGobRTR(mdlprfx + "domain.RTR.gob"); err != nil {
				log.Fatalf("%v", err)
			}
		}()

		var mapr *MAPR
		go func() {
			defer wg.Done()
			var err error
			if mapr, err = LoadGobMAPR(mdlprfx + "domain.MAPR.gob"); err != nil {
				log.Fatalf("%v", err)
			}
		}()

		var obs []int
		go func() {
			defer wg.Done()
			if _, ok := mmio.FileExists(mdlprfx + "obs"); ok {
				var err error
				if obs, err = mmio.ReadInts(mdlprfx + "obs"); err != nil {
					log.Fatalf("%v", err)
				}
			}
		}()
		wg.Wait()
		return frc, strc, rtr, mapr, obs
	}()

	return &Domain{
		Frc:  frc,
		Strc: strc,
		rtr:  rtr,
		mpr:  mpr,
		mons: mons,
		dir:  mdlprfx,
	}
}

// IsEmpty returns true if the domain has no data
func (m *Domain) IsEmpty() bool {
	return m.Strc == nil || m.mpr == nil //|| m.gd == nil
}
