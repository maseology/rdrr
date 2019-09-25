package basin

import (
	"fmt"
	"log"
	"math"

	"github.com/maseology/goHydro/hru"
)

type subsample struct {
	xr             map[int]int
	strm           map[int]float64
	ws             []hru.HRU
	in             map[int][]float64
	y, ep          [][]float64
	drel, p0       []float64
	cids, ds       []int
	fncid, dm, s0s float64
	nstep          int
}

func newSubsample(b *subdomain, p *sample, Ds, m float64, sid int) subsample {
	var pp subsample
	if sid < 0 {
		pp.cids, pp.fncid = b.cids, b.fncid
		pp.dehash(b, p, b.ncid, b.nstrm)
		pp.initialize(b.frc.Q0, Ds, m)
		return pp
	}
	if _, ok := b.rtr.swscidxr[sid]; !ok {
		log.Fatalf("subsample.newSubsample error: subwatershed id %d cannot be found.", sid)
	}
	if _, ok := p.gw[sid]; !ok {
		log.Fatalf("subsample.newSubsample error: subwatershed id %d cannot be found as a groundwater reservoir.", sid)
	}
	pp.cids, pp.fncid = b.rtr.swscidxr[sid], float64(len(b.rtr.swscidxr[sid]))
	pp.dehash(b, p, len(b.rtr.swscidxr[sid]), len(p.gw[sid].Qs))
	for _, c := range pp.cids {
		if b.rtr.sws[c] != sid {
			log.Fatalf("subsample.newSubsample error: subwatershed id %d outside of subsample.", sid)
		}
	}
	pp.initialize(b.frc.Q0, Ds, m)
	pp.ds[pp.xr[sid]] = -1 // new outlet
	return pp
}

func (pp *subsample) dehash(b *subdomain, p *sample, ncid, nstrm int) {
	pp.drel = make([]float64, ncid) // initialize mean TOPMODEL deficit
	pp.ws, pp.p0, pp.ds = make([]hru.HRU, ncid), make([]float64, ncid), make([]int, ncid)
	pp.xr = make(map[int]int, ncid) // cellID to slice id cross-reference
	pp.strm = make(map[int]float64, nstrm)
	for i, c := range pp.cids {
		sid := b.rtr.sws[c] // groundwatershed id
		pp.drel[i] = p.gw[sid].D[c]
		pp.ws[i] = *p.ws[c]
		pp.p0[i] = p.p0[c]
		pp.ds[i] = b.ds[c]
		pp.xr[c] = i
		if v, ok := p.gw[sid].Qs[c]; ok {
			pp.strm[i] = v
		}
	}
	return
}

func (pp *subsample) initialize(q0, Ds, m float64) {
	pp.dm = func() (dm float64) {
		q0t, n := 0., 0
		dm = 0. //-m * math.Log(q0/Qs)
		for {
			for c, v := range pp.strm {
				q0t += v * math.Exp((Ds-dm-pp.drel[pp.xr[c]])/m)
			}
			q0t /= pp.fncid
			if q0t <= q0 {
				break
			}
			dm += .1
			q0t = 0.
			n++
			if n > steadyiter {
				fmt.Println("steady reached max iterations")
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
