package lusg

import (
	"fmt"
	"log"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/montecarlo/invdistr"
)

// SurfGeoColl holds a collection of SurfGeo.
type SurfGeoColl map[int]SurfGeo // cell ID to SurfGeo

// LoadSurfGeo returns a pointer to a new SurfGeoColl
func LoadSurfGeo(fp string, gd *grid.Definition) *SurfGeoColl {
	fmt.Printf(" loading: %s\n", fp)
	var g grid.Indx
	g.LoadGDef(gd)
	g.NewShort(fp, false)

	// create SurfGeo collection
	p := make(map[int]SurfGeo, 8)
	for i := 1; i <= 8; i++ {
		p[i] = SurfGeo{id: i, Ksat: ksatFromID(i), kD: kDistrFromID(i)}
	}
	p[-9999] = SurfGeo{id: -9999, Ksat: ksatFromID(6), kD: kDistrFromID(6)} // unknown material

	// build collection
	m := make(map[int]SurfGeo, g.Nvalues())
	for i, v := range g.Values() {
		if x, ok := p[v]; ok {
			m[i] = x
		} else {
			log.Fatalf("no SurfGeo settings given to SurfGeo ID %d", v)
		}
	}
	sgc := SurfGeoColl(m)
	return &sgc
}

// SurfGeo holds model parameters associated with the shallow surface material properties
type SurfGeo struct {
	dK   invdistr.Trapezoid
	Ksat float64
	id   int
}

/////////////////////////////////////////////////
//// MATERIAL PROPERTIES
/////////////////////////////////////////////////

// ksatFromID returns an approximate estimate of
// hydraulic conductivity [m/s] for a given material type
func ksatFromID(sgid int) float64 {
	switch sgid {
	case 1: // Low
		return 1e-8
	case 2: // Low_Medium
		return 1e-7
	case 3: // Medium
		return 1e-6
	case 4: // Medium_High
		return 1e-5
	case 5: // High
		return 1e-4
	case 6: // Unknown (variable)
		return 1e-6
	case 7: // Streambed (alluvium/unconsolidated/fluvial/floodplain)
		return 1e-5
	case 8: // Wetland_Sediments (organics)
		return 1e-5
	default:
		log.Fatalf("ksatFromID: no value assigned to SurfGeo ID %d", sgid)
		return 0.
	}
}

// kDistrFromID returns an trapezoidal sample distribution of
// hydraulic conductivity [m/s] for a given material type
func kDistrFromID(sgid int) invdistr.Trapezoid {
	switch sgid {
	case 1: // Low
		return invdistr.NewTrapezoid(1e-11, 1e-9, 1e-7, 1e-6)
	case 2: // Low_Medium
		return invdistr.NewTrapezoid(1e-9, 1e-7, 1e-6, 1e-5)
	case 3: // Medium
		return invdistr.NewTrapezoid(1e-8, 1e-6, 1e-5, 1e-4)
	case 4: // Medium_High
		return invdistr.NewTrapezoid(1e-6, 1e-5, 1e-4, 1e-3)
	case 5: // High
		return invdistr.NewTrapezoid(1e-5, 1e-4, 1e-3, 1e-2)
	case 6: // Unknown (variable)
		return invdistr.NewTrapezoid(1e-9, 1e-7, 1e-5, 1e-3)
	case 7: // Streambed (alluvium/unconsolidated/fluvial/floodplain)
		return invdistr.NewTrapezoid(1e-8, 1e-7, 1e-5, 1e-4)
	case 8: // Wetland_Sediments (organics)
		return invdistr.NewTrapezoid(1e-8, 1e-7, 1e-5, 1e-4)
	default:
		log.Fatalf("kDistrFromID: no value assigned to SurfGeo ID %d", sgid)
		return 0.
	}
}

// syFromID returns an approximate estimate of
// specific yeild [-] for a given material type
func syFromID(sgid int) float64 {
	switch sgid {
	case 1: // Low
		return .302 // .4 (porosity)
	case 2: // Low_Medium
		return .291 // .37
	case 3: // Medium
		return .289 // .35
	case 4: // Medium_High
		return .291 // .34
	case 5: // High
		return .247 // .3
	case 6: // Unknown (variable)
		return .35 // .4
	case 7: // Streambed (alluvium/unconsolidated/fluvial/floodplain)
		return .3 // .35
	case 8: // Wetland_Sediments (organics)
		return .5 // .45
	default:
		log.Fatalf("ksatFromID: no value assigned to SurfGeo ID %d", sgid)
		return 0.
	}
}
