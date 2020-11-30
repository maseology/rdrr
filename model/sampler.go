package model

import "github.com/maseology/mmaths"

const nSmplDim = 6

func par6(u []float64) (m, hmax, slpx, dinc, soildepth, kfact float64) {
	m = mmaths.LogLinearTransform(0.01, .5, u[0])          // topmodel m -- NOTE anything less than 0.01 can lead to overflows
	hmax = mmaths.LogLinearTransform(0.001, 10., u[1])     // global surficial geology adjustment factor
	slpx = mmaths.LogLinearTransform(nugget, 3., u[2])     // cell gradient with which fcasc=1. The "range" of the gaussian variogram
	dinc = mmaths.LinearTransform(-.4, 2., u[3])           // incised stream offset
	soildepth = mmaths.LinearTransform(0., 1.5, u[4])      // depth of soilzone/ET extinction depth
	kfact = mmaths.LogLinearTransform(0.001, 10000., u[5]) // global surficial geology adjustment factor
	return
}
