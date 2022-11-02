package model

import "github.com/maseology/mmio"

func (dom *Domain) FindDm0s(lus []*Surface, mmpyr float64, cxr map[int]int, xg []int, prnt bool) []float64 {
	tt := mmio.NewTimer()
	dms := dom.EvaluateToSteadyState(mmpyr, lus, cxr, xg, prnt) //dom.FindDm0s(lus, dom.Obs.Oq[0][0]/fm3s) ///////////////////////////////////////////////////////////
	dms = dom.Drain(lus, dms, xg, 365*4)

	if prnt {
		tt.Print("inital Dm computation complete")
	}

	return dms
}

/////////////////////
// OR

// // solver
// uToDm := func(u float64) float64 { return mmaths.LinearTransform(-10., 15., u) }
// func(gi int, cids []int) {
// 	fnc := float64(len(cids)) // summations

// 	gen := func(u float64) float64 {
// 		dm, ol := uToDm(u), 0.

// 		for j := 0; j < 1e5; j++ { // steady she goes
// 			o := 0.
// 			for _, i := range cids {

// 				_, ro, _ := lus[i].Update(dm, ins[i]+p, 0.)

// 				if douts[i] { // roots and gw res discharge point
// 					o += ro * fuc[i] / fnc
// 				} else {
// 					ins[dom.Strc.DwnXR[i]] += ro
// 				}
// 			}
// 			if math.Abs(ol-o) < tolerance {
// 				// fmt.Printf("converged on %d\n", j)
// 				break
// 			}
// 			ol = o
// 		}

// 		// fmt.Printf("  %20.6f %20.6f %20.6f %20.6f\n", p, ol/fnc, u, dm)
// 		return math.Abs(p - ol/fnc)
// 	}

// 	uFib, yfib := glbopt.Fibonacci(gen)
// 	dm0[gi] = uToDm(uFib)
// 	fmt.Printf("gw %d complete %f = %f\n", gi, dm0[gi], yfib)
// }(gi, gcs)
