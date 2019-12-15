package basin

import (
	"fmt"
	"log"

	"github.com/maseology/goHydro/grid"
)

// domain holds all data and is the parent to Model
type domain struct {
	frc  *FORC            // forcing data
	strc *STRC            // structural (unchanging) data (eg, topography, solar irradiation fractions)
	mpr  *MAPR            // land use/surficial geology mapping for parameter assignment
	rtr  *RTR             // subwatershed topology
	gd   *grid.Definition // grid definition
	obs  []int            // observation cell IDs
	dir  string
}

func newDomain(ldr *Loader, buildEP bool) domain {
	frc, strc, mpr, rtr, gd, obs := ldr.load(buildEP)
	return domain{
		frc:  frc,
		strc: &strc,
		mpr:  &mpr,
		rtr:  &rtr,
		gd:   gd,
		obs:  obs,
		dir:  ldr.Dir,
	}
}

func newUniformDomain(ldr *Loader, buildEP bool) domain {
	frc, strc, mpr, rtr, gd, obs := ldr.load(buildEP)
	for i := range mpr.ilu {
		mpr.ilu[i] = -9999
		mpr.isg[i] = -9999
	}
	return domain{
		frc:  frc,
		strc: &strc,
		mpr:  &mpr,
		rtr:  &rtr,
		gd:   gd,
		obs:  obs,
		dir:  ldr.Dir,
	}
}

// ReLoadMasterForcings loads forcing data to master domain
func ReLoadMasterForcings(fp string) {
	fmt.Printf(" re-loading: %s\n", fp)
	if masterDomain.IsEmpty() {
		log.Fatalf(" ReLoadMasterForcings error: masterDomain not loaded")
	}
	masterDomain.frc, _ = loadForcing(fp, true)
}

// IsEmpty returns true if the domain has no data
func (m *domain) IsEmpty() bool {
	return m.strc == nil || m.mpr == nil || m.gd == nil
}
