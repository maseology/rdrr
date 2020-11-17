package basin

import (
	"fmt"
)

// MasterDomain holds all data from which sub-domain scale models can be derived
var masterDomain domain

// domain holds all data and is the parent to all sub models
type domain struct {
	frc  *FORC // forcing (variable) data
	strc *STRC // structural (unchanging) data (eg, topography, solar irradiation fractions)
	rtr  *RTR  // subwatershed topology
	mpr  *MAPR // land use/surficial geology mapping for parameter assignment
	// gd   *grid.Definition // grid definition
	obs []int  // observation cell IDs
	dir string // model directory
}

func newDomain(ldr *Loader) domain {
	frc, strc, mpr, rtr, _, obs := ldr.load()
	frc.q0 = avgRch // default discharge for warm-up
	return domain{
		frc:  frc,
		strc: &strc,
		mpr:  &mpr,
		rtr:  &rtr,
		// gd:   gd,
		obs: obs,
		dir: ldr.Dir,
	}
}

// IsEmpty returns true if the domain has no data
func (m *domain) IsEmpty() bool {
	return m.strc == nil || m.mpr == nil //|| m.gd == nil
}

// LoadMasterDomain loads all data from which sub-domain scale models can be derived
func LoadMasterDomain(ldr *Loader) {
	fmt.Println("Loading Master Domain..")
	masterDomain = newDomain(ldr)
}
