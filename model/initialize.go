package model

import (
	"fmt"
	"math"

	"github.com/maseology/glbopt"
	"github.com/maseology/mmaths"
)

func (pp *evaluation) initialize(Dinc, m float64, print bool) {
	smpl := func(u float64) float64 {
		return mmaths.LinearTransform(-10., 10., u)
	}
	opt := func(u []float64) float64 {
		hb := 0.
		dm := smpl(u[0])
		for i, v := range pp.strmQs {
			hb += v * math.Exp((Dinc-dm-pp.drel[i])/m)
		}
		hb /= pp.fncid
		return math.Abs(hb-avgRch) / avgRch
	}
	u, _ := glbopt.Fibonacci(opt)
	pp.dm = smpl(u)

	if print {
		fmt.Printf("intial dm = %f\n", pp.dm)
	}

	pp.s0s = 0.
	for i := 0; i < int(pp.fncid); i++ {
		pp.s0s += pp.ws[i].Storage() // initial subsample storage
	}
}

// func (pp *subsample) initialize(q0, Ds, m float64) {
// 	smpl := func(u float64) float64 {
// 		return mmaths.LinearTransform(-5., 5., u)
// 	}
// 	opt := func(u []float64) float64 {
// 		q0t, dm := 0., smpl(u[0])
// 		for c, v := range pp.strmQs {
// 			q0t += v * math.Exp((Ds-dm-pp.drel[pp.xr[c]])/m)
// 		}
// 		// for i := range pp.cids {
// 		// 	if dm < pp.drel[i] {
// 		// 		q0t -= dm + pp.drel[i]
// 		// 	}
// 		// }
// 		q0t /= pp.fncid
// 		return math.Abs(q0t-q0) / q0
// 	}
// 	u, _ := glbopt.Fibonacci(opt)
// 	pp.dm = smpl(u)

// 	pp.s0s = 0.
// 	for i := range pp.cids {
// 		pp.s0s += pp.ws[i].Storage()
// 	}
// }

// func (pp *evaluation) initialize(Dinc, m float64, print bool) {

// 	g := 0.
// 	for _, v := range pp.strmQs {
// 		g += v * math.Exp((Dinc-d)/m)
// 	}
// 	g /= float64(len(pp.strmQs))
// 	fmt.Println(len(pp.strmQs), g, avgRch, pp.fncid, math.Log(avgRch*pp.fncid))
// 	pp.dm = -m * (g + math.Log(avgRch*pp.fncid))

// 	// pp.dm = func() (dm float64) {
// 	// 	dm = -1. //0. //-m * math.Log(q0/Qs) // q0 = avgRch // default discharge for warm-up
// 	// 	if len(pp.strmQs) == 0 {
// 	// 		return
// 	// 	}
// 	// 	q0t, n := 0., 0
// 	// 	for {
// 	// 		for i, v := range pp.strmQs {
// 	// 			q0t += v * math.Exp((Dinc-dm-pp.drel[i])/m)
// 	// 		}
// 	// 		q0t /= pp.fncid
// 	// 		if q0t <= avgRch {
// 	// 			if print && dm <= 0. {
// 	// 				t := math.Abs(math.Log10(avgRch / q0t))
// 	// 				if t < 1.33 {
// 	// 					fmt.Printf("  evaluation.initialize: steady reached without iterations -- rch %.2e; Qo %.2e\n", avgRch, q0t)
// 	// 				} else {
// 	// 					fmt.Printf("  evaluation.initialize: initial discharge imposed without iteration -- rch %.2e; Qo %.2e\n", avgRch, q0t)
// 	// 				}
// 	// 			}
// 	// 			break
// 	// 		}
// 	// 		dm += .1
// 	// 		q0t = 0.
// 	// 		n++
// 	// 		if n > steadyiter {
// 	// 			if print {
// 	// 				fmt.Println("  evaluation.initialize: steady reached max iterations")
// 	// 			}
// 	// 			break
// 	// 		}
// 	// 	}
// 	// 	return
// 	// }()
// 	// pp.dm = 4.
// 	pp.s0s = 0.
// 	for i := 0; i < int(pp.fncid); i++ {
// 		pp.s0s += pp.ws[i].Storage() // initial subsample storage
// 	}
// }
