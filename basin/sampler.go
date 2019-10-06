package basin

import "github.com/maseology/mmaths"

const nSmplDim = 4

func par4(u []float64) (m, fcasc, Qs, soildepth float64) {
	m = mmaths.LogLinearTransform(0.001, .5, u[0]) // mmaths.LinearTransform(0.02, 0.06, u[0])
	fcasc = mmaths.LogLinearTransform(0.001, 10., u[1])
	Qs = mmaths.LinearTransform(-.4, 2., u[2]) // mmaths.LogLinearTransform(.001, .1, u[2])
	soildepth = mmaths.LinearTransform(0., 1., u[3])
	return
}
