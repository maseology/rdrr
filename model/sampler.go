package model

import (
	"github.com/maseology/mmaths"
	"github.com/maseology/rdrr/lusg"
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
	m = mmaths.LogLinearTransform(0.01, 5., u[0])      // topmodel m -- NOTE anything less than 0.01 can lead to overflows
	kstrm = mmaths.LinearTransform(.9, 1., u[1])       // maximum cascade fraction and given to all stream cells (~streamflow recession factor)
	mcasc = mmaths.LogLinearTransform(.001, 10., u[2]) // slope of fuzzy cascade curve
	urbDiv = mmaths.LinearTransform(0., 1., u[3])      // urban diversion: cascade fraction over urban areas routed directly to streams, remainder infiltrates
	soildepth = mmaths.LinearTransform(0., .4, u[4])   // depth of soilzone/ET extinction depth
	ksat = lusg.SurfGeoSample(u[5:])
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
