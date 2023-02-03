package rdrr

type realization struct {
	i                                           int
	ts, c, ds, incs, mons                       []int
	ins                                         [][]float64
	ya, ea, deld, drel, bo, fcasc, finf, depsto []float64
	m, dext, eafact, fngwc, d0                  float64
}
