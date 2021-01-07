package model

import (
	"log"
	"time"

	"github.com/maseology/goHydro/hru"
)

// evaluation is a dehashed model run built from a sample (this is what gets evaluated)
type evaluation struct {
	cxr                       map[int]int       // cellID to slice id cross-reference; mapping of cell id to (de-hashed) array index
	strmQs                    map[int]float64   // saturated lateral discharge (when Dm=0) at stream cells [m/ts]
	sources                   map[int][]float64 // currently: inflow from up sws
	ws                        []hru.HRU         // watershed: collection of HRUs
	t                         []time.Time       // timesteps
	y, ep                     [][]float64       // yield; demand
	drel, cascf               []float64         // relative depth to WT; cascade fraction
	ds, xrc, mxr, mt          []int             // downslope cell ID; meteo to cell xr; consecutive month index
	ca, intvl, fncid, dm, s0s float64           // cell area (m²); timestep interval (sec); float number cells; mean depth to WT; initial storage
	// carea                 float64           // catchment area (m²)
	sid, nstep int // sws ID; n timesteps
	dir        string
}

func newEvaluation(b *subdomain, p *sample, Dinc, m float64, sid int, print bool) evaluation {
	var pp evaluation
	if sid < 0 { // no subwatersheds
		log.Fatalln("evaluation.newEvaluation: legacy code, should no longer occur")
		// pp.fncid, pp.intvl = b.fncid, b.frc.IntervalSec
		// pp.dehash(b, p, sid, b.ncid, b.nstrm)
		// pp.initialize(b.frc.q0, Ds, m, print)
		// return pp
	}
	if _, ok := b.rtr.SwsCidXR[sid]; !ok {
		log.Fatalf("subsample.newSubsample error: subwatershed id %d cannot be found.", sid)
	}
	if _, ok := p.gw[sid]; !ok {
		log.Fatalf("subsample.newSubsample error: subwatershed id %d cannot be found as a groundwater reservoir.", sid)
	}
	pp.t = b.frc.T
	pp.mt = b.frc.mt
	pp.dir = p.dir
	pp.sid = sid
	pp.y, pp.ep, pp.nstep, pp.intvl, pp.ca = b.frc.D[0], b.frc.D[1], len(b.frc.T), b.frc.IntervalSec, b.strc.Acell
	// pp.carea = b.contarea
	// pp.cids, pp.fncid = b.rtr.SwsCidXR[sid], float64(len(b.rtr.SwsCidXR[sid]))
	pp.fncid = float64(len(b.rtr.SwsCidXR[sid]))
	pp.dehash(b, p, sid, len(b.rtr.SwsCidXR[sid]), len(p.gw[sid].Qs))

	// cktopo := make(map[int]bool, len(pp.cids))
	// for _, i := range pp.cids {
	// 	if _, ok := cktopo[i]; ok {
	// 		log.Fatalf(" subsample.newSubsample error: cell %d occured more than once, possible cycle", i)
	// 	}
	// 	if _, ok := b.ds[i]; !ok {
	// 		log.Fatalf(" subsample.newSubsample error: cell %d not given dowslope id", i)
	// 	}
	// 	if _, ok := cktopo[b.ds[i]]; ok {
	// 		log.Fatalf(" subsample.newSubsample error: cell %d out of topological order", i)
	// 	}
	// 	cktopo[i] = true
	// }

	pp.initialize(Dinc, m, print)
	// fmt.Printf(" **** sid: %d;  Dm0: %f;  s0: %f\n", sid, pp.dm, pp.s0s)
	pp.ds[pp.cxr[sid]] = -1 // new outlet
	return pp
}

func (pp *evaluation) dehash(b *subdomain, p *sample, sid, ncid, nstrm int) {
	pp.drel = make([]float64, ncid) // initialize mean TOPMODEL deficit
	pp.ws, pp.cascf, pp.ds = make([]hru.HRU, ncid), make([]float64, ncid), make([]int, ncid)
	// pp.f = make([][]float64, ncid)
	pp.cxr = make(map[int]int, ncid)         // cellID to slice id cross-reference
	pp.xrc = make([]int, ncid)               // cellID cross-reference
	pp.mxr = make([]int, ncid)               // met cellID to slice id cross-reference
	pp.strmQs = make(map[int]float64, nstrm) // saturated lateral discharge (when Dm=0) at stream cells [m/ts]
	for i, c := range b.rtr.SwsCidXR[sid] {
		sid := b.rtr.Sws[c] // groundwatershed id
		pp.drel[i] = p.gw[sid].D[c]
		pp.ws[i] = *p.ws[c]
		pp.cascf[i] = p.cascf[c]
		if d, ok := b.ds[c]; ok {
			pp.ds[i] = d
		} else {
			pp.ds[i] = -1 // farfield
		}
		// pp.f[i] = b.strc.f[c]
		pp.cxr[c] = i
		pp.xrc[i] = c
		pp.mxr[i] = b.frc.XR[c]
		if v, ok := p.gw[sid].Qs[c]; ok {
			pp.strmQs[i] = v
		}
	}
	return
}
