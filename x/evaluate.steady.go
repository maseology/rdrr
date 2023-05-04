package rdrr

import (
	"fmt"
	"log"
	"math"

	"github.com/maseology/goHydro/hru"
)

func (r *realization) Steady(mmpyr float64, breakyear int) {

	nc := len(r.c)
	fnc := float64(nc)
	dm, j := r.d0, 0
	x := make([]hru.Res, nc)
	for i, d := range r.depsto {
		x[i].Cap = d
	}

	ya := mmpyr / 1000. / 365.24 / 4
	ea := .1
	for {
		ssrch := 0.

		for i := range r.c { // in topological order
			in0 := x[i].Sto
			di := r.drel[i] + dm
			ro, ae, rch := 0., 0., 0.

			if di < 0. { // gw discharge
				fc := math.Exp(-di / r.m)
				if math.IsInf(fc, 0) {
					panic("evaluate(): inf")
				}
				b := fc * r.bo[i]
				ro = x[i].Overflow(b + ya)
				rch -= b //+ ea
				// ae = ea
			} else {
				// if di < r.dext {
				// 	ae = (1. - di/r.dext) * ea // linear decay
				// 	rch -= ae
				// 	ea -= ae
				// }
				ro = x[i].Overflow(ya)
			}

			ae += ea*r.eafact + x[i].Overflow(-ea*r.eafact)

			x[i].Sto += ro * (1. - r.fcasc[i])
			ro *= r.fcasc[i]

			pi := x[i].Sto * r.finf[i]
			if pi > x[i].Sto {
				rch += x[i].Sto
				x[i].Sto = 0.
			} else {
				x[i].Sto -= pi
				rch += pi
			}
			// pi := x[i].Sto * r.finf[i] // InfiltrateSurplus excess mobile water in infiltrated assuming a falling head through a unit length, returns added recharge
			// if pi > di {
			// 	x[i].Sto -= di
			// 	rch += di
			// } else {
			// 	rch += pi
			// 	x[i].Sto -= pi
			// }

			// route flows
			if r.ds[i] > -1 {
				x[r.ds[i]].Sto += ro
			}

			// test for water balance
			hruwbal := ya + in0 - x[i].Sto - ae - ro - rch
			if math.Abs(hruwbal) > nearzero {
				fmt.Printf("%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", r.i, i, hruwbal, x[i].Sto, in0, ya, ae, ro, rch)
				log.Fatalln("hru wbal error")
			}

			ssrch += rch

		}
		dm -= ssrch / r.fngwc // state update: adding recharge decreases the deficit of the gw reservoir
		if j > 365*breakyear*4 {
			// fmt.Printf(" >> steady-state not reached at %10d%14.5f%14.5f%14.5f%14.5f\n", r.i, ya, ssrch/fnc, ya+ssrch/fnc, dm)
			break
		}
		if math.Abs(ya+ssrch/fnc) < .001 {
			// fmt.Printf(" >> steady-state reached at %10d%14.5f\n", r.i, dm)
			break
		}
		j++
	}

	r.d0 = dm
}
