package opt

import "github.com/maseology/mmaths"

func Par5(u []float64) (m, acasc, maxFcasc, soildepth, dinc float64) {
	m = mmaths.LinearTransform(0., 10., u[0]) // topmodel m -- NOTE anything less than 0.01 can lead to overflows [m]
	acasc = mmaths.LogLinearTransform(.0001, .1, u[1])
	maxFcasc = mmaths.LinearTransform(.8, 1., u[2])
	soildepth = mmaths.LinearTransform(0., 2., u[3]) // depth of soilzone/ET extinction depth [m]
	dinc = mmaths.LinearTransform(-1., 1., u[4])
	//
	// grdMin = mmaths.LogLinearTransform(.00001, 1., u[1]) // minium slope given a cascade fraction
	// mcasc = mmaths.LogLinearTransform(.001, 10., u[2])   // slope of fuzzy cascade curve
	// // kstrm = mmaths.LinearTransform(0., 1., u[3])         // maximum cascade fraction and given to all stream cells (~streamflow recession factor)
	// // soildepth = mmaths.LinearTransform(0., 1.5, u[4])
	return
}
