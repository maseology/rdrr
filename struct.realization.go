package rdrr

import (
	"math"

	"github.com/maseology/goHydro/hru"
)

type realization struct {
	x []hru.Res
	// spr, sae, sro, srch   []float64
	drel, bo, finf, fcasc []float64
	cids, cds, cmon       []int
	eaf, dextm, fnc, fgnc float64
	nc                    int
}

func (r *realization) rdrr(ya, ea, dmm float64, m, j, k int) (qmon []float64, qout, dm float64) {
	// ssae, ssro, ssdsto := 0., 0., 0. // needed for WATERBALANCE below
	ssnetrch := 0.
	qmon = make([]float64, len(r.cmon))
	for i, c := range r.cids {
		avail := ea
		// dsto0 := r.x[i].Sto       // needed for WATERBALANCE below
		xs, ae, rch := 0., 0., 0. //, gwd
		dim := r.drel[i] + dmm

		// gw discharge, including evaporation from gw reservoir
		if dim < 0. {
			fc := math.Exp(-dim)

			// if math.IsInf(fc, 0) { // keep m>.01
			// 	panic("evaluate(): inf")
			// 	// fc = 1000.
			// }

			b := fc * r.bo[i]            // groundwater flux to cell
			xs = r.x[i].Overflow(b + ya) // excess (potential runoff)
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
			xs = r.x[i].Overflow(ya)
		}

		// evaporate from detention/surface storage
		if avail > 0. {
			ae += avail + r.x[i].Overflow(-avail)
		}

		// Infiltrate surplus/excess mobile water in infiltrated assuming a falling head through a unit length, returns added recharge
		// rch += xs * r.finf[i]
		// ro := xs * (1. - r.finf[i])
		pi := r.x[i].Sto * r.finf[i]
		r.x[i].Sto -= pi
		rch += pi
		ro := xs

		// cascade portion of storage
		// if dim <= 0. { // if land is saturated assume max cascade
		// 	// 	r.x[i].Sto += ro * (1. - r.maxcasc)
		// 	// 	ro *= r.maxcasc
		// 	// } else {
		r.x[i].Sto += ro * (1. - r.fcasc[i])
		ro *= r.fcasc[i]
		// }

		// route flows
		if ids := r.cds[i]; ids > -1 { // FUTURE CHANGE: change r.cds to a list of pointers to downslope hrus like done with r.rte
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

		// // test for WATERBALANCE
		// hruwbal := ya + dsto0 - r.x[i].Sto - ae - ro - rch
		// if math.Abs(hruwbal) > nearzero {
		// 	fmt.Printf("%10d%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, i, hruwbal, r.x[i].Sto, dsto0, ya, ae, ro, rch)
		// 	panic("hru wbal error")
		// }

		// r.spr[m*r.nc+i] += ya
		// r.sae[m*r.nc+i] += ae
		// r.sro[m*r.nc+i] += ro
		// r.srch[m*r.nc+i] += rch
		// // r.sgwd[m*r.nc+i] += gwd
		ssnetrch += rch //- gwd

		// // needed for WATERBALANCE below
		// ssae += ae
		// ssro += ro
		// ssdsto += r.x[i].Sto - dsto0
	}

	// // per timestep subwatershed WATERBALANCE
	// swswbal := ya - (ssae+ssro+ssnetrch+ssdsto)/r.fnc
	// if math.Abs(swswbal) > nearzero {
	// 	fmt.Printf("%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, swswbal, ssdsto, ya, ssae, ssro, ssnetrch)
	// 	panic("sws t wbal error")
	// }

	return qmon, qout, -ssnetrch / r.fgnc // sws outflow; state update: adding recharge decreases the deficit of the gw reservoir
}
