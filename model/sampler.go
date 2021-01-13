package model

import "github.com/maseology/mmaths"

const nSmplDim = 7

func Par7(u []float64) (m, grdMin, kstrm, mcasc, soildepth, kfact, dinc float64) {
	m = mmaths.LogLinearTransform(0.1, 500., u[0]) // topmodel m -- NOTE anything less than 0.01 can lead to overflows
	grdMin = mmaths.LogLinearTransform(.00001, 1., u[1])
	kstrm = mmaths.LinearTransform(0., 1., u[2])            // maximum cascade fraction and given to all stream cells (~streamflow recession factor)
	mcasc = mmaths.LogLinearTransform(.001, 10., u[3])      // slope of fuzzy cascade curve
	soildepth = mmaths.LinearTransform(0., 1.5, u[4])       // depth of soilzone/ET extinction depth
	kfact = mmaths.LogLinearTransform(0.0001, 10000., u[5]) // global surficial geology adjustment factor
	dinc = mmaths.LinearTransform(0., 2., u[6])
	return
}

// func Par4(u []float64) (m, grng, soildepth, kfact float64) {
// 	m = mmaths.LogLinearTransform(0.1, 500., u[0])          // topmodel m -- NOTE anything less than 0.01 can lead to overflows
// 	grng = mmaths.LogLinearTransform(nugget, 3., u[1])      // cell gradient with which fcasc=1. The "range" of the gaussian variogram
// 	soildepth = mmaths.LinearTransform(0., 1.5, u[2])       // depth of soilzone/ET extinction depth
// 	kfact = mmaths.LogLinearTransform(0.0001, 10000., u[3]) // global surficial geology adjustment factor
// 	return
// }

// func par6(u []float64) (m, hmax, grng, dinc, soildepth, kfact float64) {
// 	m = mmaths.LogLinearTransform(0.01, 5., u[0])          // topmodel m -- NOTE anything less than 0.01 can lead to overflows
// 	hmax = mmaths.LogLinearTransform(0.001, 10., u[1])     // global surficial geology adjustment factor
// 	grng = mmaths.LogLinearTransform(nugget, 3., u[2])     // cell gradient with which fcasc=1. The "range" of the gaussian variogram
// 	dinc = mmaths.LinearTransform(-.4, 2., u[3])           // incised stream offset
// 	soildepth = mmaths.LinearTransform(0., 1.5, u[4])      // depth of soilzone/ET extinction depth
// 	kfact = mmaths.LogLinearTransform(0.001, 10000., u[5]) // global surficial geology adjustment factor
// 	return
// }
