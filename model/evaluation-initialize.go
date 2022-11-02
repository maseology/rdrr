package model

func (pp *evaluation) initialize(print bool) {
	// smpl := func(u float64) float64 {
	// 	return mmaths.LinearTransform(-100., 10., u)
	// }
	// opt := func(u []float64) float64 {
	// 	hb := 0.
	// 	dm := smpl(u[0])
	// 	for i, v := range pp.strmQs {
	// 		hb += v * math.Exp((Dinc-dm-pp.drel[i])/m)
	// 	}
	// 	hb /= pp.fncid
	// 	return math.Abs(hb-avgRch) / avgRch
	// }
	// u, _ := glbopt.Fibonacci(opt)
	// pp.dm = smpl(u)
	// if print {
	// 	fmt.Printf(" initial dm = %f\n", pp.dm)
	// }
	ms := 0.
	for c := range pp.strmQs {
		ms += pp.m[pp.gxr[c]]
	}
	pp.dm = ms / float64(len(pp.strmQs))

	pp.s0s = 0.
	for i := 0; i < int(pp.fncid); i++ {
		pp.s0s += pp.ws[i].Storage() // initial subsample storage
	}
}
