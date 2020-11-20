package lusg

import (
	"log"
)

// surface types
const (
	Noflow = iota
	Waterbody
	ShortVegetation
	TallVegetation
	Urban
	Agriculture
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
func (l *LandUse) Rebuild1(soildepth, fimp, ifct float64) (rzsto, surfsto float64) {
	return func() (rzsto, surfsto float64) {
		rzsto = soildepth * l.Porosity * (1. - l.Fc)
		surfsto = soildepth*l.Porosity*l.Fc + fimp*l.DepSto + l.IntSto*ifct
		switch l.ID {
		case Waterbody, Channel, Lake: // Open water
			rzsto = soildepth
			surfsto = 0.
		case Noflow:
			rzsto = 0.
			surfsto = 1000.
		case ShortVegetation, TallVegetation, Forest, Swamp, Wetland, SparseVegetation, DenseVegetation, Agriculture, Meadow, Marsh, Urban, Barren:
			// do nothing
		default:
			log.Fatalf(" LandUse.Rebuild1: no value assigned to ID %d", l.ID)
		}
		return
	}()
}
