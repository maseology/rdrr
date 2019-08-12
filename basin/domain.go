package basin

import (
	"fmt"
	"log"

	"github.com/maseology/goHydro/grid"
)

// MasterDomain holds all data from which sub-domain scale models can be derived
var masterDomain domain

// domain holds all data and is the parent to Model
type domain struct {
	frc  *FORC            // forcing data
	strc *STRC            // structural (unchanging) data (eg, topography, solar irradiation fractions)
	mpr  *MAPR            // land use/surficial geology mapping for parameter assignment
	gd   *grid.Definition // grid definition
}

// LoadMasterDomain loads all data from which sub-domain scale models can be derived
func LoadMasterDomain(ldr *Loader) {
	fmt.Println("Loading Master Domain..")
	masterDomain = newDomain(ldr)
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

func newDomain(ldr *Loader) domain {
	frc, strc, mpr, gd := ldr.load()
	return domain{
		frc:  frc,
		strc: &strc,
		mpr:  &mpr,
		gd:   gd,
	}
}

func newUniformDomain(ldr *Loader) domain {
	frc, strc, mpr, gd := ldr.load()
	for i := range mpr.ilu {
		mpr.ilu[i] = -9999
		mpr.isg[i] = -9999
	}
	return domain{
		frc:  frc,
		strc: &strc,
		mpr:  &mpr,
		gd:   gd,
	}
}
