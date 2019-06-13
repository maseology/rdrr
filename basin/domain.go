package basin

import (
	"github.com/maseology/goHydro/grid"
)

// domain holds all data and is the parent to Model
type domain struct {
	frc  *FORC            // forcing data
	strc *STRC            // structural (unchanging) data (eg, topography, solar irradiation fractions)
	mpr  *MAPR            // land use/surficial geology mapping for parameter assignment
	gd   *grid.Definition // grid definition
}

func newDomain(ldr *Loader) domain {
	frc, strc, mpr, gd := ldr.load()
	return domain{
		frc:  &frc,
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
		frc:  &frc,
		strc: &strc,
		mpr:  &mpr,
		gd:   gd,
	}	
}
