package lusg

import (
	"fmt"
	"log"

	"github.com/maseology/goHydro/grid"
)

const (
	defaultDepSto    = 0.001  // [m]
	defaultIntSto    = 0.0005 // [m]
	defaultSoilDepth = 0.3    // [m]
	defaultPorosity  = 0.3    // [-]
	defaultFc        = 0.1    // [-]
)

// LandUseColl holds a collection of LandUse.
type LandUseColl map[int]LandUse

// LoadLandUse returns a pointer to a new LandUseColl
func LoadLandUse(fp string, gd *grid.Definition) *LandUseColl {
	fmt.Printf(" loading: %s\n", fp)
	var g grid.Indx
	g.LoadGDef(gd)
	g.NewShort(fp, false)

	// create LandUse collection
	p := make(map[int]LandUse, 8)
	for _, i := range g.UniqueValues() {
		sz, dp, f := propsFromSOLRIS(i)
		p[i] = LandUse{id: i, DrnSto: sz, SrfSto: dp, Fimp: f}
	}

	// build collection
	m := make(map[int]LandUse, g.Nvalues())
	for i, v := range g.Values() {
		var lut = p[v]
		m[i] = lut
	}
	luc := LandUseColl(m)
	return &luc
}

// LandUse holds model parameters associated with land use/cover
type LandUse struct {
	DrnSto, SrfSto, Fimp float64
	id                   int
}

/////////////////////////////////////////////////
//// LAND USE PROPERTIES
/////////////////////////////////////////////////

// propsFromSOLRIS returns landuse properties from a given
// SOLRIS ID. (rootzone/drainable storage, surface storage, fimp)
func propsFromSOLRIS(id int) (rzsto, surfsto, fimp float64) {
	rzsto, surfsto, fimp = defaultSoilDepth*defaultPorosity*(1.-defaultFc), defaultSoilDepth*defaultPorosity*defaultFc, 0.
	switch id {
	case 201: // Transportation
		fimp = 1.0
	case 202: // Built Up Area - Pervious
		surfsto += defaultIntSto / 2.
	case 203: // Built Up Area - Impervious
		fimp = 0.9
		surfsto += fimp*defaultDepSto + (1.-fimp)*defaultIntSto
	case 193, 250: // "Undifferentiated", but really Agriculture
		surfsto += defaultIntSto
	case 23, 43: // Treed Sand Dune, Treed Cliff and Talus (canopy but little to no ground cover)
		surfsto += defaultIntSto
	case 51: // Open Alvar (85% bare)
		surfsto += 0.15 * defaultIntSto
	case 52, 53: // Shrub Alvar, Treed Alvar (canopy with partial ground cover/85% bare)
		surfsto += 1.15 * defaultIntSto
	case 83, 90, 91, 92, 93, 131, 135, 191, 192: // tall vegetation, vegetated ground cover
		surfsto += defaultIntSto * 2.
	case 82: // partial tall vegetation, vegetated ground cover
		surfsto += defaultIntSto * 1.5
	case 140, 150, 160: // wetlands/marshes
		surfsto += defaultIntSto
	case 81: // short vegetation
		surfsto += defaultIntSto
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
