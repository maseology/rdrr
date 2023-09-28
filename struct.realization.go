package rdrr

import (
	"fmt"
	"log"
	"math"

	"github.com/maseology/goHydro/hru"
)

type realization struct {
	x                     []hru.Res
	drel, bo, finf, fcasc []float64
	spr, sae, sro, srch   []float64
	cids, sds             []int
	eaf, dextm, fnc, fgnc float64 // m,
	cmon                  int
}

func (r *realization) rdrr(ya, ea, dmm float64, j, k int) (float64, float64) {
	qout, ssae, ssro, ssrch, ssdsto := 0., 0., 0., 0., 0.
	for i := range r.cids {

		avail := ea
		dsto0 := r.x[i].Sto
		ro, ae, rch := 0., 0., 0.
		dim := r.drel[i] + dmm

		// gw discharge, including evaporation from gw reservoir
		if dim < 0. {
			fc := math.Exp(-dim)
			if math.IsInf(fc, 0) { // keep m>.01
				panic("evaluate(): inf")
				// fc = 1000.
			}
			b := fc * r.bo[i]
			ro = r.x[i].Overflow(b + ya)
			rch -= b + avail*r.eaf
			ae = avail * r.eaf
			avail -= ae
		} else {
			if dim < r.dextm {
				ae = (1. - dim/r.dextm) * avail // linear decay
				rch -= ae
				avail -= ae
			}
			ro = r.x[i].Overflow(ya)
		}

		// Infiltrate surplus/excess mobile water in infiltrated assuming a falling head through a unit length, returns added recharge
		pi := r.x[i].Sto * r.finf[i]
		r.x[i].Sto -= pi
		rch += pi

		// evaporate from detention storage
		if avail > 0. {
			ae += avail + r.x[i].Overflow(-avail)
		}

		r.x[i].Sto += ro * (1. - r.fcasc[i])
		ro *= r.fcasc[i]

		// route flows
		if ids := r.sds[i]; ids > -1 {
			r.x[ids].Sto += ro
		} else {
			qout += ro
		}

		// test for water balance
		hruwbal := ya + dsto0 - r.x[i].Sto - ae - ro - rch
		if math.Abs(hruwbal) > nearzero {
			fmt.Printf("%10d%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, i, hruwbal, r.x[i].Sto, dsto0, ya, ae, ro, rch)
			log.Fatalln("hru wbal error")
		}

		ssae += ae
		ssro += ro
		ssrch += rch
		ssdsto += r.x[i].Sto - dsto0

		r.spr[i] += ya
		r.sae[i] += ae
		r.sro[i] += ro
		r.srch[i] += rch
	}

	// per timestep subwatershed waterbalance
	swswbal := ya - (ssae+ssro+ssrch+ssdsto)/r.fnc
	if math.Abs(swswbal) > nearzero {
		fmt.Printf("%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, swswbal, ssdsto, ya, ssae, ssro, ssrch)
		log.Fatalln("sws t wbal error")
	}
	return qout, -ssrch / r.fgnc // sws outflow; state update: adding recharge decreases the deficit of the gw reservoir
}
