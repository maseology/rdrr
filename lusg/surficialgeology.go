package lusg

import (
	"log"
	"math"

	"github.com/maseology/mmaths"
	"github.com/maseology/montecarlo/invdistr"
	"github.com/maseology/montecarlo/jointdist"
)

const (
	low = iota + 1
	lowMedium
	medium
	mediumHigh
	high
	unknown   // variable
	streambed // fluvial/floodplain
	wetlandSediments
)

// SurfGeoColl holds a collection of SurfGeo.
type SurfGeoColl map[int]SurfGeo // cell ID to SurfGeo

// LoadSurfGeo returns a pointer to a new SurfGeoColl
func LoadSurfGeo(UniqueValues []int) *SurfGeoColl {
	// create SurfGeo collection
	p := make(map[int]SurfGeo, len(UniqueValues))
	for _, id := range UniqueValues {
		switch id {
		case -9999, -1, 0:
			p[id] = SurfGeo{
				ID:   id,
				Ksat: ksatFromID(6),
				SY:   syFromID(6),
				dK:   ksatTrapDistrFromID(6),
				dP:   porDistrFromID(6),
			}
		default:
			p[id] = SurfGeo{
				ID:   id,
				Ksat: ksatFromID(id),
				SY:   syFromID(id),
				dK:   ksatTrapDistrFromID(id),
				dP:   porDistrFromID(id),
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

// Sample returns a sample from the SurfGeo's range
func Sample(u []float64) []float64 {
	k := make([]float64, 8)
	f := func(sgid int) (float64, float64) {
		switch sgid {
		case 1: // Low
			return 1e-11, 1e-6
			// return buildLogTrapezoid(1e-11, 1e-9, 1e-7, 1e-6)
		case 2: // Low_Medium
			return 1e-9, 1e-5
			// return buildLogTrapezoid(1e-9, 1e-7, 1e-6, 1e-5)
		case 3: // Medium
			return 1e-8, 1e-4
			// return buildLogTrapezoid(1e-8, 1e-6, 1e-5, 1e-4)
		case 4: // Medium_High
			return 1e-6, 1e-3
			// return buildLogTrapezoid(1e-6, 1e-5, 1e-4, 1e-3)
		case 5: // High
			return 1e-5, 1e-2
			// return buildLogTrapezoid(1e-5, 1e-4, 1e-3, 1e-2)
		case 6: // Unknown (variable)
			return 1e-9, 1e-3
			// return buildLogTrapezoid(1e-9, 1e-7, 1e-5, 1e-3)
		case 7: // Streambed (alluvium/unconsolidated/fluvial/floodplain)
			return 1e-8, 1e-4
			// return buildLogTrapezoid(1e-8, 1e-7, 1e-5, 1e-4)
		case 8: // Wetland_Sediments (organics)
			return 1e-8, 1e-4
			// return buildLogTrapezoid(1e-8, 1e-7, 1e-5, 1e-4)
		default:
			log.Fatalf("Sample: no value assigned to SurfGeo ID %d", sgid)
			return 0., 0.
		}
	}
	for i := 0; i < 8; i++ {
		l, h := f(i)
		k[i] = mmaths.LogLinearTransform(l, h, u[i])
	}
	return k
}

// SampleTrapezoid returns a sample from the SurfGeo's range using trapezoidal distributions
func SampleTrapezoid(u []float64) []float64 {
	k := make([]float64, 8)
	for i := 0; i < 8; i++ {
		k[i] = ksatTrapDistrFromID(i + 1).P(u[i])
	}
	return k
}

// SampleNested returns a nested sample from the SurfGeo's range
func SampleNested(u []float64) []float64 {
	k := make([]float64, 8)
	for i, un := range jointdist.Nested(u[:5]...) {
		k[4-i] = mmaths.LogLinearTransform(1e-11, 1e-3, un) // low through high
	}
	k[5] = ksatTrapDistrFromID(unknown).P(u[5])          // unknown/variable
	k[6] = ksatTrapDistrFromID(streambed).P(u[6])        // streambed
	k[7] = ksatTrapDistrFromID(wetlandSediments).P(u[7]) // wetland
	return k
}

// func (s *SurfGeo) Sample(u ...float64) (ksat, por, sy float64) {
// 	ksat = s.dK.P(u[0])
// 	por = s.dP.P(u[1])
// 	sy = s.SY
// 	return
// }

/////////////////////////////////////////////////
//// MATERIAL PROPERTIES
/////////////////////////////////////////////////

// ksatFromID returns an approximate estimate of
// saturated hydraulic conductivity [m/s] for a given material type
func ksatFromID(sgid int) float64 {
	switch sgid {
	case 1: // Low
		return 1e-9
	case 2: // Low_Medium
		return 1e-8 // ~316 mm/yr
	case 3: // Medium
		return 1e-7
	case 4: // Medium_High
		return 1e-6
	case 5: // High
		return 1e-5
	case 6: // Unknown (variable)
		return 1e-8
	case 7: // Streambed (alluvium/unconsolidated/fluvial/floodplain)
		return 1e-5
	case 8: // Wetland_Sediments (organics)
		return 1e-6
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

// ksatTrapDistrFromID returns a trapezoidal sample distribution of hydraulic conductivity [m/s] for a given material type
func ksatTrapDistrFromID(sgid int) *invdistr.Map {
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
		log.Fatalf("syFromID: no value assigned to SurfGeo ID %d", sgid)
		return 0.
	}
}
