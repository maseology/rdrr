package model

import "github.com/maseology/mmaths"

const nSmplDim = 5

func par5(u []float64) (m, smax, dinc, soildepth, kfact float64) {
	m = mmaths.LogLinearTransform(0.001, .5, u[0])        // topmodel m
	smax = mmaths.LogLinearTransform(0.001, 10., u[1])    // cell slope with which p0=1.
	dinc = mmaths.LinearTransform(-.4, 2., u[2])          // incised stream offset
	soildepth = mmaths.LinearTransform(0., 1.5, u[3])     // depth of soilzone/ET extinction depth
	kfact = mmaths.LogLinearTransform(0.01, 10000., u[4]) // global surficial geology adjustment factor
	return
}
