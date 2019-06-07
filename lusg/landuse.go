package lusg

import (
	"log"
)

// LandUseColl holds a collection of LandUse.
type LandUseColl map[int]LandUse

// LoadLandUse returns a pointer to a new LandUseColl
func LoadLandUse(UniqueValues []int) *LandUseColl {
	// create LandUse collection
	p := make(map[int]LandUse, len(UniqueValues))
	for _, i := range UniqueValues {
		_, _, f, t := defaultsFromSOLRIS(i)
		p[i] = LandUse{id: i, Fimp: f, Intfct: t}
	}

	luc := LandUseColl(p)
	return &luc
}

// LandUse holds model parameters associated with land use/cover
type LandUse struct {
	Fimp, Intfct float64
	id           int
}

/////////////////////////////////////////////////
//// LAND USE PROPERTIES
/////////////////////////////////////////////////

// GetDefaultsSOLRIS returns default SOLRIS landuse properties
// from a given LandUse struct. (rootzone/drainable storage, surface storage, fimp, interception factor)
func (l *LandUse) GetDefaultsSOLRIS() (rzsto, surfsto, fimp, ifct float64) {
	return defaultsFromSOLRIS(l.id)
}

// defaultsFromSOLRIS returns landuse properties from a given default
// SOLRIS ID. (rootzone/drainable storage, surface storage, fimp)
func defaultsFromSOLRIS(id int) (rzsto, surfsto, fimp, ifct float64) {
	const (
		defaultDepSto    = 0.001  // [m]
		defaultIntSto    = 0.0005 // [m]
		defaultSoilDepth = 0.3    // [m]
		defaultPorosity  = 0.3    // [-]
		defaultFc        = 0.1    // [-]
	)
	return buildFromSOLRIS(defaultSoilDepth, defaultPorosity, defaultFc, defaultIntSto, defaultDepSto, id)
}

// buildFromSOLRIS returns landuse properties from a given
// SOLRIS ID. (rootzone/drainable storage, surface storage, fimp)
func buildFromSOLRIS(soildepth, porosity, fc, intsto, depsto float64, id int) (rzsto, surfsto, fimp, ifct float64) {
	rzsto, surfsto, fimp, ifct = soildepth*porosity*(1.-fc), soildepth*porosity*fc, 0., 0.
	switch id {
	case 201: // Transportation
		fimp = 1.
	case 202: // Built Up Area - Pervious
		surfsto += intsto / 2.
		ifct = 0.5
	case 203: // Built Up Area - Impervious
		fimp = 0.9
		surfsto += fimp*depsto + (1.-fimp)*intsto
		ifct = 1.
	case 193, 250: // "Undifferentiated", but really Agriculture
		surfsto += intsto
		ifct = 1.
	case 23, 43: // Treed Sand Dune, Treed Cliff and Talus (canopy but little to no ground cover)
		surfsto += intsto
		ifct = 1.
	case 51: // Open Alvar (85% bare)
		ifct = 0.15
		surfsto += ifct * intsto
	case 52, 53: // Shrub Alvar, Treed Alvar (canopy with partial ground cover/85% bare)
		ifct = 1.15
		surfsto += ifct * intsto
	case 83, 90, 91, 92, 93, 131, 135, 191, 192: // tall vegetation, vegetated ground cover
		surfsto += intsto * 2.
		ifct = 2.
	case 82: // partial tall vegetation, vegetated ground cover
		surfsto += intsto * 1.5
		ifct = 1.5
	case 140, 150, 160: // wetlands/marshes
		surfsto += intsto
		ifct = 1.
	case 81: // short vegetation
		surfsto += intsto
		ifct = 1.
	case 170: // Open water
		rzsto = 0.
		surfsto = 0.
	case 11, 21, 41, 204, 205, -9999: // bare (no vegetation)
	// do nothing
	default:
		log.Fatalf("propsFromSOLRIS: no value asigned to SOLRIS ID %d", id)
	}
	return
	// 11. Open Beach/Bar
	// 21. Open Sand Dune
	// 23. Treed Sand Dune
	// 41. Open Cliff and Talus
	// 43. Treed Cliff and Talus
	// 51. Open Alvar
	// 52. Shrub Alvar
	// 53. Treed Alvar
	// 81. Open Tallgrass Prairie
	// 82. Tallgrass Savannah
	// 83. Tallgrass Woodland
	// 90. Forest
	// 91. Coniferous Forest
	// 92. Mixed Forest
	// 93. Deciduous Forest
	// 131. Treed Swamp
	// 135. Thicket Swamp
	// 140. Fen
	// 150. Bog
	// 160. Marsh
	// 170. Open Water
	// 191. Plantation
	// 192. Hedge Row
	// 193. Tilled
	// 201. Transportation
	// 202. Built Up Area - Pervious
	// 203. Built Up Area - Impervious
	// 204. Extraction - Aggregate
	// 205. Extraction - Peat/Topsoil
	// 250. Undifferentiated
}
