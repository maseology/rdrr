package model

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/maseology/goHydro/hru"
)

// evaluation is a dehashed model run built from a sample (this is what gets evaluated)
type evaluation struct {
	cxr                          map[int]int       // cellID to slice id cross-reference; mapping of cell id to (de-hashed) array index
	strmQs                       map[int]float64   // saturated lateral discharge (when Dm=0) at stream cells [m/ts]
	sources                      map[int][]float64 // currently: inflow from up sws
	ws                           []hru.HRU         // watershed: collection of HRUs
	t                            []time.Time       // timesteps
	y, ep                        [][]float64       // yield; demand
	drel, cascf                  []float64         // relative depth to WT; cascade fraction
	ds, mxr, mt                  []int             // downslope cell ID; meteo to cell xr; consecutive month index
	carea, intvl, fncid, dm, s0s float64           // catchment area (m²); timestep interval (sec); float number cells; mean depth to WT; initial storage
	nstep                        int               // n timesteps
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
	pp.y, pp.ep, pp.nstep, pp.intvl, pp.carea = b.frc.D[0], b.frc.D[1], len(b.frc.T), b.frc.IntervalSec, b.contarea
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

	pp.initialize(b.frc.q0, Dinc, m, print)
	// fmt.Printf(" **** sid: %d;  Dm0: %f;  s0: %f\n", sid, pp.dm, pp.s0s)
	pp.ds[pp.cxr[sid]] = -1 // new outlet
	return pp
}

func (pp *evaluation) dehash(b *subdomain, p *sample, sid, ncid, nstrm int) {
	pp.drel = make([]float64, ncid) // initialize mean TOPMODEL deficit
	pp.ws, pp.cascf, pp.ds = make([]hru.HRU, ncid), make([]float64, ncid), make([]int, ncid)
	// pp.f = make([][]float64, ncid)
	pp.cxr = make(map[int]int, ncid)         // cellID to slice id cross-reference
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
		pp.mxr[i] = b.frc.XR[c]
		if v, ok := p.gw[sid].Qs[c]; ok {
			pp.strmQs[i] = v
		}
	}
	return
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

func (pp *evaluation) initialize(q0, Dinc, m float64, print bool) {
	pp.dm = func() (dm float64) {
		dm = 0. //-m * math.Log(q0/Qs)
		if len(pp.strmQs) == 0 {
			return
		}
		q0t, n := 0., 0
		for {
			for i, v := range pp.strmQs {
				q0t += v * math.Exp((Dinc-dm-pp.drel[i])/m)
			}
			q0t /= pp.fncid
			if q0t <= q0 {
				if print && dm <= 0. {
					fmt.Println("  evaluation.initialize: steady reached without iterations")
				}
				break
			}
			dm += .1
			q0t = 0.
			n++
			if n > steadyiter {
				if print {
					fmt.Println("  evaluation.initialize: steady reached max iterations")
				}
				break
			}
		}
		return
	}()
	pp.s0s = 0.
	for i := 0; i < int(pp.fncid); i++ {
		pp.s0s += pp.ws[i].Storage() // initial subsample storage
	}
}
