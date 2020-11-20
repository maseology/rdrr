package model

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/mmio"
)

// MasterDomain holds all data from which sub-domain scale models can be derived
var masterDomain domain

// domain holds all data and is the parent to all sub models
type domain struct {
	frc  *FORC  // forcing (variable) data
	strc *STRC  // structural (unchanging) data (eg, topography, solar irradiation fractions)
	rtr  *RTR   // subwatershed topology
	mpr  *MAPR  // land use/surficial geology mapping for parameter assignment
	obs  []int  // observation cell IDs
	dir  string // model directory/prefix
}

// LoadMasterDomain loads all data from which sub-domain scale models can be derived
func LoadMasterDomain(mdlprfx, obsfp string) {
	fmt.Println("Loading Master Domain..")
	masterDomain = func() domain {
		frc, strc, rtr, mpr, obs := func() (*FORC, *STRC, *RTR, *MAPR, []int) {
			var wg sync.WaitGroup
			wg.Add(5)
			var frc *FORC
			go func() {
				defer wg.Done()
				var err error
				if frc, err = LoadGobFORC(mdlprfx + "FORC.gob"); err != nil {
					log.Fatalf("%v", err)
				}
			}()

			var strc *STRC
			go func() {
				defer wg.Done()
				var err error
				if strc, err = LoadGobSTRC(mdlprfx + "STRC.gob"); err != nil {
					log.Fatalf("%v", err)
				}
			}()

			var rtr *RTR
			go func() {
				defer wg.Done()
				var err error
				if rtr, err = LoadGobRTR(mdlprfx + "RTR.gob"); err != nil {
					log.Fatalf("%v", err)
				}
			}()

			var mapr *MAPR
			go func() {
				defer wg.Done()
				var err error
				if mapr, err = LoadGobMAPR(mdlprfx + "MAPR.gob"); err != nil {
					log.Fatalf("%v", err)
				}
			}()

			var obs []int
			go func() {
				defer wg.Done()
				var err error
				if obs, err = mmio.ReadInts(obsfp); err != nil {
					log.Fatalf("%v", err)
				}
			}()
			wg.Wait()
			return frc, strc, rtr, mapr, obs
		}()
		frc.q0 = avgRch // default discharge for warm-up
		return domain{
			frc:  frc,
			strc: strc,
			rtr:  rtr,
			mpr:  mpr,
			obs:  obs,
			dir:  mdlprfx,
		}
	}()
}

// IsEmpty returns true if the domain has no data
func (m *domain) IsEmpty() bool {
	return m.strc == nil || m.mpr == nil //|| m.gd == nil
}
