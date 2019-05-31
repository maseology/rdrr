package basin

import (
	"log"
	"math"
	"sync"

	"github.com/maseology/rdrr/lusg"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
)

func (b *subdomain) buildSfrac(f1 float64) map[int]float64 {
	fc := make(map[int]float64, len(b.cids))
	for _, c := range b.cids {
		fc[c] = math.Min(f1*b.strc.t.TEC[c].S, 1.)
	}
	return fc
}

func (b *subdomain) toDefaultSample(rill, m, n float64) sample {
	var wg sync.WaitGroup

	ts := b.frc.h.IntervalSec()
	if ts <= 0. {
		log.Fatalf("toDefaultSample error, timestep (IntervalSec) = %v", ts)
	}
	ws := make(hru.WtrShd, b.ncid)
	var gw gwru.TMQ
	assignHRUs := func() {
		defer wg.Done()
		var recurs func(int)
		recurs = func(cid int) {
			var ll, gg int
			var ok bool
			if ll, ok = b.mpr.ilu[cid]; !ok {
				log.Fatalf("toDefaultSample.assignHRUs error, no LandUse assigned to cell ID %d", cid)
			}
			if gg, ok = b.mpr.isg[cid]; !ok {
				log.Fatalf("toDefaultSample.assignHRUs error, no SurfGeo assigned to cell ID %d", cid)
			}
			var lu lusg.LandUse
			var sg lusg.SurfGeo
			if lu, ok = b.mpr.lu[ll]; !ok {
				log.Fatalf("toDefaultSample.assignHRUs error, no LandUse assigned of type %d", ll)
			}
			if sg, ok = b.mpr.sg[gg]; !ok {
				log.Fatalf("toDefaultSample.assignHRUs error, no SurfGeo assigned to type %d", gg)
			}

			var h hru.HRU
			drnsto, srfsto, fimp, _ := lu.GetDefaultsSOLRIS()
			h.Initialize(drnsto, srfsto, fimp, sg.Ksat, ts)
			ws[cid] = &h
			for _, upcid := range b.strc.t.UpIDs(cid) {
				recurs(upcid)
			}
		}
		recurs(b.cid0)
	}
	buildTopmodel := func() {
		defer wg.Done()
		ksat := make(map[int]float64)
		var recurs func(int)
		recurs = func(cid int) {
			if gg, ok := b.mpr.isg[cid]; ok {
				if sg, ok := b.mpr.sg[gg]; ok {
					ksat[cid] = sg.Ksat * ts // [m/ts]
					for _, upcid := range b.strc.t.UpIDs(cid) {
						recurs(upcid)
					}
				} else {
					log.Fatalf("toDefaultSample.buildTopmodel error, no SurfGeo assigned to type %d", gg)
				}
			} else {
				log.Fatalf("toDefaultSample.buildTopmodel error, no SurfGeo assigned to cell ID %d", cid)
			}
		}
		recurs(b.cid0)

		if b.frc.Q0 <= 0. {
			log.Fatalf("toDefaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.Q0)
		}
		medQ := b.frc.Q0 * b.strc.a * float64(len(ksat)) // [m/d] to [m³/d]
		gw.New(ksat, b.strc.t, b.strc.w, medQ, 2*medQ, m)
	}

	wg.Add(2)
	go assignHRUs()
	go buildTopmodel()
	wg.Wait()

	na := make(map[int]float64, b.ncid)
	for i := range ws {
		na[i] = n
	}
	return sample{
		ws:   ws,
		gw:   gw,
		p0:   b.buildC0(na, ts), // b.buildSfrac(f1),
		p1:   b.buildC2(na, ts),
		rill: rill,
	}
}
