package rdrr

import (
	"math"

	"github.com/maseology/goHydro/hru"
)

type basin struct {
	x                          []hru.Res
	drel, bo, finf, fcasc, eaf []float64
	cids, ads                  []int
	fnc, fgnc                  float64 // , dextm
	nc                         int
	coll                       *basinCollect
}

type basinCollect struct{ spr, sae, sro, srch, sdch []float64 }

func (r *basin) rdrr(ya, ea, dmm float64, mnt, j, k int) (qout, dm float64) {
	// ssae, ssro, ssdsto := 0., 0., 0. // needed for WATERBALANCE below
	ssnetrch := 0.
	for i := range r.cids {
		avail := ea
		xs, ae, rch, gwd := 0., 0., 0., 0.
		dim := r.drel[i] + dmm
		// dsto0 := r.x[i].Sto // needed for WATERBALANCE below

		if dim < 0. { // gw discharge, including evaporation from gw reservoir
			fc := math.Exp(-dim)

			// if math.IsInf(fc, 0) { // keep m>.01
			// 	panic("evaluate(): inf")
			// 	// fc = 1000.
			// }

			b := fc * r.bo[i] // groundwater flux to cell
			// rch -= b + avail*r.eaf       // evaporation from saturated lands (net recharge)
			gwd += b + avail*r.eaf[i] // evaporation from saturated lands
			ae = avail * r.eaf[i]     // evaporation
			avail -= ae
			// r.x[i].Sto += b + ya // add to cell storage
			xs = r.x[i].Overflow(b + ya)
		} else {
			// if dim < r.dextm {
			// 	ae = (1. - dim/r.dextm) * avail // linear decay
			// 	// rch -= ae
			// 	gwd += ae
			// 	avail -= ae
			// }
			// r.x[i].Sto += ya
			xs = r.x[i].Overflow(ya)
		}

		// evaporate from detention/surface storage
		if avail > 1e-5 {
			// ae += avail + r.x[i].Overflow(-avail)
			pe := avail * math.Min(r.x[i].DegreeSat(), 1.)
			ae += pe + r.x[i].Overflow(-pe)
		}

		// Infiltrate surplus/excess mobile water in infiltrated assuming a falling head through a unit length, returns added recharge
		pi := r.x[i].Sto * r.finf[i]
		r.x[i].Sto -= pi
		rch += pi
		// xs = r.x[i].Overflow(0.) // excess (potential runoff)

		// cascade portion of surplus/excess mobile water
		r.x[i].Sto += xs * (1. - r.fcasc[i])
		ro := xs * r.fcasc[i]

		// route flows
		if ids := r.ads[i]; ids > -1 {
			r.x[ids].Sto += ro
		} else {
			qout += ro
		}

		// // test for WATERBALANCE
		// hruwbal := ya + dsto0 - r.x[i].Sto - ae - ro - rch + gwd
		// if math.Abs(hruwbal) > nearzero {
		// 	fmt.Printf("%10d%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, i, hruwbal, r.x[i].Sto, dsto0, ya, ae, ro, rch, gwd)
		// 	panic("hru wbal error")
		// }

		if r.coll != nil {
			r.coll.spr[mnt*r.nc+i] += ya
			r.coll.sae[mnt*r.nc+i] += ae
			r.coll.sro[mnt*r.nc+i] += ro
			r.coll.srch[mnt*r.nc+i] += rch
			r.coll.sdch[mnt*r.nc+i] += gwd
		}

		ssnetrch += rch - gwd

		// // needed for WATERBALANCE below
		// ssae += ae
		// ssro += ro
		// ssdsto += r.x[i].Sto - dsto0
	}

	// // per timestep subwatershed WATERBALANCE
	// swswbal := ya - (ssae+ssro+ssnetrch+ssdsto)/r.fnc
	// if math.Abs(swswbal) > nearzero {
	// 	fmt.Printf("%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, swswbal, ssdsto, ya, ssae, ssro, ssnetrch)
	// 	panic("hrus wbal error")
	// }

	return qout, -ssnetrch / r.fgnc // sws outflow; state update: adding recharge decreases the deficit of the gw reservoir
}
