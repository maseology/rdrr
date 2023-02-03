package rdrr

import (
	"fmt"
	"log"
	"math"

	"github.com/maseology/goHydro/hru"
)

const nbins = 12

type result struct {
	i           int
	ae, ro, rch [][nbins]float64
	q, d        []float64
	dmlast      float64
	mons        map[int][]float64
}

func (r *realization) rdrr() result {
	// tt := time.Now()

	nc, nt := len(r.c), len(r.ya)
	fnc := float64(nc)
	sae, sro, srch := make([][nbins]float64, nc), make([][nbins]float64, nc), make([][nbins]float64, nc)
	// tae, tro, trch := make([]float64, nt), make([]float64, nt), make([]float64, nt)
	hyd := make([]float64, nt)
	dm := r.d0

	x := make([]hru.Res, nc)
	for i, d := range r.depsto {
		x[i].Cap = d
	}

	mon := make(map[int][]float64)
	for _, m := range r.mons {
		mon[m] = make([]float64, nt)
	}

	for j, mj := range r.ts {
		dm += r.deld[j]
		ya := r.ya[j]
		ea := r.ea[j]
		ssae, ssro, ssrch, ssdsto := 0., 0., 0., 0.

		for i, inc := range r.incs {
			x[inc].Sto += r.ins[i][j]
		}

		for i, c := range r.c { // in topological order

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
				rch -= b + ea
				ae = ea
				ea = 0.
			} else {
				if di < r.dext {
					ae = (1. - di/r.dext) * ea // linear decay
					rch -= ae
					ea -= ae
				}
				ro = x[i].Overflow(ya)
			}

			x[i].Sto += ro * (1. - r.fcasc[i])
			ro *= r.fcasc[i]

			pi := x[i].Sto * r.finf[i] // InfiltrateSurplus excess mobile water in infiltrated assuming a falling head through a unit length, returns added recharge
			if pi > di {
				x[i].Sto -= di
				rch += di
			} else {
				rch += pi
				x[i].Sto -= pi
			}

			ae += ea*r.eafact + x[i].Overflow(-ea*r.eafact)

			// route flows
			if r.ds[i] > -1 {
				x[r.ds[i]].Sto += ro
			} else {
				hyd[j] = ro
			}
			if _, ok := mon[c]; ok {
				mon[c][j] = ro
			}

			// test for water balance
			hruwbal := ya + in0 - x[i].Sto - ae - ro - rch
			if math.Abs(hruwbal) > nearzero {
				fmt.Printf("%10d%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", r.i, j, i, hruwbal, x[i].Sto, in0, ya, ae, ro, rch)
				log.Fatalln("hru wbal error")
			}

			ssae += ae
			ssro += ro
			ssrch += rch
			ssdsto += x[i].Sto - in0

			sae[i][mj] += ae
			sro[i][mj] += ro
			srch[i][mj] += rch

			// tae[j] += ae
			// tro[j] += ro
			// trch[j] += rch
		}

		dd := -ssrch / r.fngwc // state update: adding recharge decreases the deficit of the gw reservoir
		r.deld[j] += dd        // update
		dm += dd               // state update: adding recharge decreases the deficit of the gw reservoir

		swswbal := ya - (ssae+ssro+ssrch+ssdsto)/fnc
		if math.Abs(swswbal) > nearzero {
			fmt.Printf("%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", r.i, j, swswbal, ssdsto, ya, ssae, ssro, ssrch)
			log.Fatalln("sws t wbal error")
		}
	}

	// sub-watershed water budgeting
	// writeFloats(fmt.Sprintf("wtbdgt-ae-%d.bin", r.i), tae)
	// writeFloats(fmt.Sprintf("wtbdgt-ro-%d.bin", r.i), tro)
	// writeFloats(fmt.Sprintf("wtbdgt-rch-%d.bin", r.i), trch)

	// fmt.Printf("  %d elapsed: %v\n", r.i, time.Since(tt))
	return result{r.i, sae, sro, srch, hyd, r.deld, dm, mon}
}
