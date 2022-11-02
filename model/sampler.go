package model

import (
	"log"
	"math"

	"github.com/maseology/mmaths"
	"github.com/maseology/montecarlo/invdistr"
)

const nDefltSmplDim = 7
const nSGeoSmplDim = 13

func par7(u []float64) (m, grdMin, kstrm, mcasc, soildepth, kfact, dinc float64) {
	m = mmaths.LogLinearTransform(0.1, 500., u[0])       // topmodel m -- NOTE anything less than 0.01 can lead to overflows
	grdMin = mmaths.LogLinearTransform(.00001, 1., u[1]) // minium slope given a cascade fraction
	kstrm = mmaths.LinearTransform(0., 1., u[2])         // maximum cascade fraction and given to all stream cells (~streamflow recession factor)
	mcasc = mmaths.LogLinearTransform(.001, 10., u[3])   // slope of fuzzy cascade curve
	// urbDiv = mmaths.LogLinearTransform(0., 1., u[4])        // urban diversion: cascade fraction over urban areas routed directly to streams, remainder infiltrates
	soildepth = mmaths.LinearTransform(0., 1.5, u[4])       // depth of soilzone/ET extinction depth
	kfact = mmaths.LogLinearTransform(0.0001, 10000., u[5]) // global surficial geology adjustment factor
	dinc = mmaths.LinearTransform(0., 2., u[6])
	return
}

func parSurfGeo(u []float64) (m, kstrm, mcasc, urbDiv, soildepth float64, ksat []float64) {
	m = mmaths.LogLinearTransform(0.01, 5., u[0])        // topmodel m -- NOTE anything less than 0.01 can lead to overflows
	kstrm = 1. - mmaths.LinearTransform(.0001, .1, u[1]) // maximum cascade fraction and given to all stream cells (~streamflow recession factor)
	mcasc = mmaths.LogLinearTransform(.01, 10., u[2])    // slope of fuzzy cascade curve
	urbDiv = mmaths.LinearTransform(0., 1., u[3])        // urban diversion: cascade fraction over urban areas routed directly to streams, remainder infiltrates
	soildepth = mmaths.LinearTransform(0., .4, u[4])     // depth of soilzone/ET extinction depth
	ksat = SurfGeoSample(u[5:])
	return
}

// func parSurfGeo(u []float64) (m, grdMin, kstrm, mcasc, urbDiv, soildepth, dinc float64, ksat []float64) {
// 	m = mmaths.LogLinearTransform(0.01, 5., u[0])        // topmodel m -- NOTE anything less than 0.01 can lead to overflows
// 	grdMin = mmaths.LogLinearTransform(.00001, 1., u[1]) // gradient under which no flow will cascade
// 	kstrm = mmaths.LinearTransform(.9, 1., u[2])         // maximum cascade fraction and given to all stream cells (~streamflow recession factor)
// 	mcasc = mmaths.LogLinearTransform(.001, 10., u[3])   // slope of fuzzy cascade curve
// 	urbDiv = mmaths.LinearTransform(0., 1., u[4])        // urban diversion: cascade fraction over urban areas routed directly to streams, remainder infiltrates
// 	soildepth = mmaths.LinearTransform(0., 1.5, u[5])    // depth of soilzone/ET extinction depth
// 	dinc = mmaths.LinearTransform(0., 2., u[6])
// 	ksat = lusg.SurfGeoSample(u[7:])
// 	return
// }

func SurfGeoSample(u []float64) []float64 {
	k := make([]float64, 8)
	f := func(sgid int) *invdistr.Map {
		switch sgid {
		case 1: // Low
			// return buildLogLinear(1e-11, 1e-6)
			return buildLogTriangularOM(1.10e-08)
			// return buildLogTrapezoid(1e-11, 1e-9, 1e-7, 1e-6)
		case 2: // Low_Medium
			return buildLogLinear(1e-9, 1e-5)
			// return buildLogTrapezoid(1e-9, 1e-7, 1e-6, 1e-5)
		case 3: // Medium
			return buildLogLinear(1e-8, 1e-4)
			// return buildLogTrapezoid(1e-8, 1e-6, 1e-5, 1e-4)
		case 4: // Medium_High
			return buildLogLinear(1e-6, 1e-3)
			// return buildLogTrapezoid(1e-6, 1e-5, 1e-4, 1e-3)
		case 5: // High
			return buildLogLinear(1e-5, 1e-2)
			// return buildLogTrapezoid(1e-5, 1e-4, 1e-3, 1e-2)
		case 6: // Unknown (variable)
			return buildLogLinear(1e-9, 1e-3)
			// return buildLogTrapezoid(1e-9, 1e-7, 1e-5, 1e-3)
		case 7: // Streambed (alluvium/unconsolidated/fluvial/floodplain)
			return buildLogLinear(1e-9, 1e-6)
			// return buildLogTrapezoid(1e-8, 1e-7, 1e-5, 1e-4)
		case 8: // Wetland_Sediments (organics)
			return buildLogLinear(1e-8, 1e-4)
			// return buildLogTrapezoid(1e-8, 1e-7, 1e-5, 1e-4)
		default:
			log.Fatalf("Sample: no value assigned to SurfGeo ID %d", sgid)
			return nil
		}
	}
	for i := 0; i < 8; i++ {
		// l, h := f(i + 1)
		// k[i] = mmaths.LogLinearTransform(l, h, u[i])
		k[i] = f(i + 1).P(u[i])
	}
	return k
}

func buildLogLinear(l, h float64) *invdistr.Map {
	if l > h {
		log.Panicf("sampler-buildLogLinear error: invalid arguments l, h = %v, %v\n", l, h)
	}
	return &invdistr.Map{
		Low:   math.Log10(l),
		High:  math.Log10(h),
		Log:   true,
		Distr: &invdistr.Uniform{},
	}
}

func buildLogTriangularOM(m float64) *invdistr.Map {
	return buildLogTriangular(m/10., m, m*10.)
}
func buildLogTriangular(l, m, h float64) *invdistr.Map {
	if l > m || m > h || l < 0. {
		log.Panicf("sampler-buildLogTriangular error: invalid arguments l, m, h = %v, %v, %v\n", l, m, h)
	}
	l10 := math.Log10(l)
	m10 := math.Log10(m)
	h10 := math.Log10(h)
	return &invdistr.Map{
		Low:   l10,
		High:  h10,
		Log:   true,
		Distr: invdistr.NewTriangle((m10 - l10) / (h10 - l10)),
	}

}
