package basin

import (
	"log"
	"math"
	"sync"

	"github.com/maseology/rdrr/lusg"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
)

const secperday = 86400.0

func (b *subdomain) buildSfrac(f1 float64) map[int]float64 {
	fc := make(map[int]float64, len(b.cids))
	for _, c := range b.cids {
		fc[c] = math.Min(f1*math.Sqrt(b.strc.t.TEC[c].S), 1.)
	}
	return fc
}

func (b *subdomain) toDefaultSample(fQ0, m, fcasc float64) sample {
	var wg sync.WaitGroup

	ts := b.frc.h.IntervalSec()
	if ts <= 0. {
		log.Fatalf("toDefaultSample error, timestep (IntervalSec) = %v", ts)
	}
	ws := make(hru.WtrShd, b.ncid)
	var gw map[int]*gwru.TMQ

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
		printHRUprops(ws)
	}

	buildTopmodel := func() {
		defer wg.Done()
		swsid := make(map[int][]int, len(b.mpr.sws)) // id'd by outlet cell (typically a stream cell)
		for k, v := range b.mpr.sws {
			if _, ok := swsid[v]; !ok {
				swsid[v] = []int{k}
			} else {
				swsid[v] = append(swsid[v], k)
			}
		}
		gw = make(map[int]*gwru.TMQ, len(swsid))
		ksatC, tiC, gC := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid)
		for k, v := range swsid {
			ksat := make(map[int]float64)
			var recurs func(int)
			recurs = func(cid int) {
				if gg, ok := b.mpr.isg[cid]; ok {
					if sg, ok := b.mpr.sg[gg]; ok {
						ksat[cid] = sg.Ksat * ts // [m/ts]
						for _, upcid := range b.strc.t.UpIDs(cid) {
							if _, ok := swsid[upcid]; !ok { // not including outlet/stream cells
								recurs(upcid)
							}
						}
					} else {
						log.Fatalf("toDefaultSample.buildTopmodel error, no SurfGeo assigned to type %d", gg)
					}
				} else {
					log.Fatalf("toDefaultSample.buildTopmodel error, no SurfGeo assigned to cell ID %d", cid)
				}
			}
			recurs(k)

			if len(ksat) != len(v) {
				log.Fatalf("toDefaultSample.buildTopmodel topology error")
			}
			if b.frc.Q0 <= 0. {
				log.Fatalf("toDefaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.Q0)
			}
			medQ := b.frc.Q0 // [m/ts] * b.strc.a * float64(len(ksat)) // [m/ts] to [m³/ts]

			var gwt gwru.TMQ
			ti, g := gwt.New(ksat, b.strc.t, b.strc.w, medQ, fQ0*medQ, m)
			for i, k := range ksat {
				ksatC[i] = k
				tiC[i] = ti[i]
				gC[i] = g
			}
			gw[k] = &gwt
		}
		saveBinaryMap1(tiC, "tmq.topographic_index.rmap")
		saveBinaryMap1(gC, "tmq.gamma.rmap")
		saveBinaryMap1(ksatC, "tmq.ksat_mpts.rmap")
	}

	wg.Add(2)
	go assignHRUs()
	go buildTopmodel()
	wg.Wait()

	// // assumes uniform rounghness
	// ns := make(map[int]float64, b.ncid)
	// for i := range ws {
	// 	ns[i] = n
	// }

	return sample{
		ws: ws,
		gw: gw,
		p0: b.buildSfrac(fcasc),
		// p0: b.buildC0(ns, ts), // ,
		// p1: b.buildC2(ns, ts),
	}
}
