package basin

import (
	"fmt"
	"log"
	"math"

	"github.com/maseology/goHydro/hru"
)

type subsample struct {
	cxr            map[int]int // mapping of cell to index
	strm           map[int]float64
	ws             []hru.HRU
	in             map[int][]float64
	t              []temporal
	y, ep          [][]float64
	drel, p0       []float64
	cids, ds, mxr  []int
	fncid, dm, s0s float64
	nstep          int
	// f              [][]float64 // solar irradiation coefficient/(adjusted) potential evaporation
}

func newSubsample(b *subdomain, p *sample, Ds, m float64, sid int, print bool) subsample {
	var pp subsample
	if sid < 0 {
		pp.cids, pp.fncid = b.cids, b.fncid
		pp.dehash(b, p, b.ncid, b.nstrm)
		pp.initialize(b.frc.Q0, Ds, m, print)
		return pp
	}
	if _, ok := b.rtr.swscidxr[sid]; !ok {
		log.Fatalf("subsample.newSubsample error: subwatershed id %d cannot be found.", sid)
	}
	if _, ok := p.gw[sid]; !ok {
		log.Fatalf("subsample.newSubsample error: subwatershed id %d cannot be found as a groundwater reservoir.", sid)
	}
	pp.t = b.frc.t
	pp.cids, pp.fncid = b.rtr.swscidxr[sid], float64(len(b.rtr.swscidxr[sid]))
	pp.dehash(b, p, len(b.rtr.swscidxr[sid]), len(p.gw[sid].Qs))

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

	pp.initialize(b.frc.Q0, Ds, m, print)
	// fmt.Printf(" **** sid: %d;  Dm0: %f;  s0: %f\n", sid, pp.dm, pp.s0s)
	pp.ds[pp.cxr[sid]] = -1 // new outlet
	return pp
}

func (pp *subsample) dehash(b *subdomain, p *sample, ncid, nstrm int) {
	pp.drel = make([]float64, ncid) // initialize mean TOPMODEL deficit
	pp.ws, pp.p0, pp.ds = make([]hru.HRU, ncid), make([]float64, ncid), make([]int, ncid)
	// pp.f = make([][]float64, ncid)
	pp.cxr = make(map[int]int, ncid) // cellID to slice id cross-reference
	pp.mxr = make([]int, ncid)       // met cellID to slice id cross-reference
	pp.strm = make(map[int]float64, nstrm)
	for i, c := range pp.cids {
		sid := b.rtr.sws[c] // groundwatershed id
		pp.drel[i] = p.gw[sid].D[c]
		pp.ws[i] = *p.ws[c]
		pp.p0[i] = p.p0[c]
		pp.ds[i] = b.ds[c]
		// pp.f[i] = b.strc.f[c]
		pp.cxr[c] = i
		pp.mxr[i] = b.frc.x[c]
		if v, ok := p.gw[sid].Qs[c]; ok {
			pp.strm[i] = v
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
// 		for c, v := range pp.strm {
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

func (pp *subsample) initialize(q0, Ds, m float64, print bool) {
	pp.dm = func() (dm float64) {
		q0t, n := 0., 0
		dm = 0. //-m * math.Log(q0/Qs)
		for {
			for i, v := range pp.strm {
				q0t += v * math.Exp((Ds-dm-pp.drel[i])/m)
			}
			q0t /= pp.fncid
			if q0t <= q0 {
				if print && dm <= 0. {
					fmt.Println("subsample.initialize: steady reached without iterations")
				}
				break
			}
			dm += .1
			q0t = 0.
			n++
			if n > steadyiter {
				fmt.Println("subsample.initialize: steady reached max iterations")
				break
			}
		}
		return
	}()
	pp.s0s = 0.
	for i := range pp.cids {
		pp.s0s += pp.ws[i].Storage()
	}
}
