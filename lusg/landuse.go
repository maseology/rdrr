package lusg

import (
	"log"
	"math"
)

// surface types
const (
	Noflow = iota
	Waterbody
	ShortVegetation
	TallVegetation
	Urban
	Agriculture // 5
	Forest
	Meadow
	Wetland
	Swamp
	Marsh // 10
	Channel
	Lake
	Barren
	SparseVegetation
	DenseVegetation
)

// LandUseColl holds a collection of LandUse.
type LandUseColl map[int]LandUse

// LandUse holds model parameters associated with land use/cover
// rootzone/drainable storage; surface storage; fimp
type LandUse struct {
	// Fimp, Intfct float64
	DepSto, IntSto, SoilDepth, Porosity, Fc float64
	// RZsto, Surfsto, Fimp float64
	ID int
}

// Rebuild1 returns default landuse properties, but with soildepth specified
// from a given LandUse struct. (rootzone/drainable storage, surface storage)
func (l *LandUse) Rebuild1(soildepth, fimp, ifct float64) (rzsto, surfsto, sma0, srf0 float64) {
	return func() (rzsto, surfsto, sma0, srf0 float64) {
		sma0, srf0 = 0., 0.
		rzsto = soildepth * l.Porosity * (1. - l.Fc)
		surfsto = soildepth*l.Porosity*l.Fc + fimp*l.DepSto + l.IntSto*ifct
		switch l.ID {
		case Channel:
			// rzsto = 0.
			surfsto = 0.
		case Waterbody, Lake: // Open water
			rzsto = 0.
			surfsto = soildepth
			srf0 = soildepth
		case Noflow:
			rzsto = 0.
			surfsto = math.MaxFloat64
		case Urban: // (assumed drained)
			rzsto *= (1. - fimp)
			surfsto = soildepth*l.Porosity*l.Fc*(1.-fimp) + fimp*l.DepSto + l.IntSto*ifct
		case ShortVegetation, TallVegetation, Forest, Swamp, Wetland, SparseVegetation, DenseVegetation, Agriculture, Meadow, Marsh, Barren:
			// do nothing
		default:
			log.Fatalf(" LandUse.Rebuild1: no value assigned to ID %d", l.ID)
		}
		return
	}()
}
