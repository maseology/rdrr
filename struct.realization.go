package rdrr

import (
	"math"

	"github.com/maseology/goHydro/hru"
)

type realization struct {
	x                     []hru.Res
	spr, sae, sro, srch   [][12]float64
	drel, bo, finf, fcasc []float64
	cids, cds, cmon       []int
	rte                   SWStopo
	eaf, dextm, fnc, fgnc float64
}

func (r *realization) rdrr(ya, ea, dmm float64, m, j, k int) (qmon []float64, qout, dm float64) {
	// ssae, ssro, ssdsto := 0., 0., 0.
	ssnetrch := 0.
	qmon = make([]float64, len(r.cmon))
	for i, c := range r.cids {
		avail := ea
		// dsto0 := r.x[i].Sto
		ro, ae, rch := 0., 0., 0. //, gwd
		dim := r.drel[i] + dmm

		// gw discharge, including evaporation from gw reservoir
		if dim < 0. {
			fc := math.Exp(-dim)

			// if math.IsInf(fc, 0) { // keep m>.01
			// 	panic("evaluate(): inf")
			// 	// fc = 1000.
			// }

			b := fc * r.bo[i]            // groundwater flux to cell
			ro = r.x[i].Overflow(b + ya) // runoff
			rch -= b + avail*r.eaf       // evaporation from saturated lands
			// gwd += b + avail*r.eaf // evaporation from saturated lands
			ae = avail * r.eaf
			avail -= ae
		} else {
			if dim < r.dextm {
				ae = (1. - dim/r.dextm) * avail // linear decay
				rch -= ae
				// gwd += ae
				avail -= ae
			}
			ro = r.x[i].Overflow(ya)
		}

		// evaporate from detention/surface storage
		if avail > 0. {
			ae += avail + r.x[i].Overflow(-avail)
		}

		// Infiltrate surplus/excess mobile water in infiltrated assuming a falling head through a unit length, returns added recharge
		pi := r.x[i].Sto * r.finf[i]
		r.x[i].Sto -= pi
		rch += pi

		// cascade portion of storage
		r.x[i].Sto += ro * (1. - r.fcasc[i])
		ro *= r.fcasc[i]

		// route flows
		if ids := r.cds[i]; ids > -1 {
			r.x[ids].Sto += ro
		} else {
			qout += ro
		}

		// grab monitor
		for i, cm := range r.cmon {
			if c == cm {
				qmon[i] = ro
			}
		}

		// // test for water balance
		// hruwbal := ya + gwd + dsto0 - r.x[i].Sto - ae - ro - rch
		// if math.Abs(hruwbal) > nearzero {
		// 	fmt.Printf("%10d%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, i, hruwbal, r.x[i].Sto, dsto0, ya, gwd, ae, ro, rch)
		// 	panic("hru wbal error")
		// }

		r.spr[i][m] += ya
		r.sae[i][m] += ae
		r.sro[i][m] += ro
		r.srch[i][m] += rch
		// r.sgwd[i][m] += gwd
		ssnetrch += rch //- gwd

		// ssae += ae
		// ssro += ro
		// ssdsto += r.x[i].Sto - dsto0
	}

	// // per timestep subwatershed waterbalance
	// swswbal := ya - (ssae+ssro+ssnetrch+ssdsto)/r.fnc
	// if math.Abs(swswbal) > nearzero {
	// 	fmt.Printf("%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, swswbal, ssdsto, ya, ssae, ssro, ssnetrch)
	// 	panic("sws t wbal error")
	// }

	return qmon, qout, -ssnetrch / r.fgnc // sws outflow; state update: adding recharge decreases the deficit of the gw reservoir
}
