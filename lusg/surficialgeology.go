package lusg

import (
	"log"
	"math"

	"github.com/maseology/montecarlo/invdistr"
)

// SurfGeoColl holds a collection of SurfGeo.
type SurfGeoColl map[int]SurfGeo // cell ID to SurfGeo

// LoadSurfGeo returns a pointer to a new SurfGeoColl
func LoadSurfGeo(UniqueValues []int) *SurfGeoColl {
	// create SurfGeo collection
	p := make(map[int]SurfGeo, len(UniqueValues))
	for _, i := range UniqueValues {
		if i == -9999 { // unknown material
			p[-9999] = SurfGeo{
				ID:   -9999,
				Ksat: ksatFromID(6),
				SY:   syFromID(6),
				dK:   ksatDistrFromID(6),
				dP:   porDistrFromID(6),
			}
		} else {
			p[i] = SurfGeo{
				ID:   i,
				Ksat: ksatFromID(i),
				SY:   syFromID(i),
				dK:   ksatDistrFromID(i),
				dP:   porDistrFromID(i),
			}
		}
	}

	sgc := SurfGeoColl(p)
	return &sgc
}

// SurfGeo holds model parameters associated with the shallow surface material properties
type SurfGeo struct {
	dK, dP   *invdistr.Map
	Ksat, SY float64
	ID       int
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

func buildLogTrapezoid(l, m, n, h float64) *invdistr.Map {
	if l > m || m > n || n > h {
		log.Panicf("lsug.buildLogTrapezoid error: invalid arguments l, m, n, h = %v, %v, %v, %v\n", l, m, n, h)
	}
	l10 := math.Log10(l)
	m10 := math.Log10(m)
	n10 := math.Log10(n)
	h10 := math.Log10(h)
	return &invdistr.Map{
		Low:   l10,
		High:  h10,
		Log:   true,
		Distr: invdistr.NewTrapezoid((m10-l10)/(h10-l10), (n10-l10)/(h10-l10), 2., 2.),
	}
}

func buildLinear(l, h float64) *invdistr.Map {
	if l > h {
		log.Panicf("lsug.buildLinear error: invalid arguments l, h = %v, %v\n", l, h)
	}
	return &invdistr.Map{
		Low:   l,
		High:  h,
		Log:   false,
		Distr: &invdistr.Uniform{},
	}
}

// ksatDistrFromID returns a trapezoidal sample distribution of
// hydraulic conductivity [m/s] for a given material type
func ksatDistrFromID(sgid int) *invdistr.Map {
	switch sgid {
	case 1: // Low
		return buildLogTrapezoid(1e-11, 1e-9, 1e-7, 1e-6)
	case 2: // Low_Medium
		return buildLogTrapezoid(1e-9, 1e-7, 1e-6, 1e-5)
	case 3: // Medium
		return buildLogTrapezoid(1e-8, 1e-6, 1e-5, 1e-4)
	case 4: // Medium_High
		return buildLogTrapezoid(1e-6, 1e-5, 1e-4, 1e-3)
	case 5: // High
		return buildLogTrapezoid(1e-5, 1e-4, 1e-3, 1e-2)
	case 6: // Unknown (variable)
		return buildLogTrapezoid(1e-9, 1e-7, 1e-5, 1e-3)
	case 7: // Streambed (alluvium/unconsolidated/fluvial/floodplain)
		return buildLogTrapezoid(1e-8, 1e-7, 1e-5, 1e-4)
	case 8: // Wetland_Sediments (organics)
		return buildLogTrapezoid(1e-8, 1e-7, 1e-5, 1e-4)
	default:
		log.Fatalf("kDistrFromID: no value assigned to SurfGeo ID %d", sgid)
		return nil
	}
}

// porDistrFromID returns a linear sample distribution of
// porosity [-] for a given material type
func porDistrFromID(sgid int) *invdistr.Map {
	switch sgid {
	case 1: // clay
		return buildLinear(0.4, 0.7)
	case 2, 3, 6, 7, 8: // loam/silt
		return buildLinear(0.35, 0.5)
	case 4, 5: // sand
		return buildLinear(0.25, 0.5)
	default:
		return nil
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

/////////////////////////////////////////////////
//// MATERIAL PROPERTIES
/////////////////////////////////////////////////

// Sample returns a sample from the SurfGeo's range
func (s *SurfGeo) Sample(u ...float64) (ksat, por, sy float64) {
	ksat = s.dK.P(u[0])
	por = s.dP.P(u[1])
	sy = s.SY
	return
}
