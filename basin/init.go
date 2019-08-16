package basin

import (
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/maseology/rdrr/lusg"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
)

const secperday = 86400.

func (b *subdomain) buildSfrac(fcasc float64) map[int]float64 {
	fc := make(map[int]float64, len(b.cids))
	for _, c := range b.cids {
		fc[c] = math.Min(fcasc*math.Sqrt(b.strc.t.TEC[c].S), 1.)
	}
	return fc
}

func (b *subdomain) toDefaultSample(Qo, m, fcasc float64) sample {
	var wg sync.WaitGroup

	ts := b.frc.h.IntervalSec() // [s/ts]
	if ts <= 0. {
		log.Fatalf(" toDefaultSample error, timestep (IntervalSec) = %v", ts)
	}
	ws := make(hru.WtrShd, b.ncid)
	var gw map[int]*gwru.TMQ
	var swsr, celr map[int]float64

	assignHRUs := func() {
		defer fmt.Println(" assignHRUs complete")
		defer wg.Done()
		var recurs func(int)
		recurs = func(cid int) {
			var ll, gg int
			var ok bool
			if ll, ok = b.mpr.ilu[cid]; !ok {
				log.Fatalf(" toDefaultSample.assignHRUs error, no LandUse assigned to cell ID %d", cid)
			}
			if gg, ok = b.mpr.isg[cid]; !ok {
				log.Fatalf(" toDefaultSample.assignHRUs error, no SurfGeo assigned to cell ID %d", cid)
			}
			var lu lusg.LandUse
			var sg lusg.SurfGeo
			if lu, ok = b.mpr.lu[ll]; !ok {
				log.Fatalf(" toDefaultSample.assignHRUs error, no LandUse assigned of type %d", ll)
			}
			if sg, ok = b.mpr.sg[gg]; !ok {
				log.Fatalf(" toDefaultSample.assignHRUs error, no SurfGeo assigned to type %d", gg)
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
		// printHRUprops(ws)
	}

	buildTopmodel := func() {
		defer fmt.Println(" buildTopmodel complete")
		defer wg.Done()
		// // sws := make(map[int]int, b.ncid)
		// // if len(b.mpr.sws) > 0 {
		// // 	osws := b.mpr.sws[outlet]
		// // 	for _, cid := range b.cids {
		// // 		if i, ok := b.mpr.sws[cid]; ok {
		// // 			if i == osws {
		// // 				sws[cid] = outlet // crops sws to outlet
		// // 			} else {
		// // 				sws[cid] = i
		// // 			}
		// // 		} else {
		// // 			sws[cid] = cid // main channel outlet cells
		// // 		}
		// // 	}
		// // } else { // entire model domain is one subwatershed to outlet
		// // 	for _, cid := range b.cids {
		// // 		sws[cid] = outlet
		// // 	}
		// // }
		// swscidxr := make(map[int][]int, b.ncid) // id'd by outlet cell (typically a stream cell)
		// osws := b.mpr.sws[b.cid0]
		// for _, cid := range b.cids {
		// 	// if i, ok := b.mpr.sws[cid]; ok {
		// 	// 	swsid := i
		// 	// 	if i == osws {
		// 	// 		swsid = b.cid0 // crops sws to outlet
		// 	// 	}
		// 	// } else {

		// 	// }

		// 	swsid := b.mpr.sws[cid]
		// 	if swsid == osws {
		// 		swsid = b.cid0
		// 	}
		// 	if _, ok := swscidxr[swsid]; !ok {
		// 		swscidxr[swsid] = []int{cid}
		// 	} else {
		// 		swscidxr[swsid] = append(swscidxr[swsid], cid)
		// 	}
		// }
		swscidxr := make(map[int][]int, len(b.rtr.sws)) // id'd by outlet cell (typically a stream cell)
		for k, v := range b.rtr.sws {
			if _, ok := swscidxr[v]; !ok {
				swscidxr[v] = []int{k}
			} else {
				swscidxr[v] = append(swscidxr[v], k)
			}
		}
		gw = make(map[int]*gwru.TMQ, len(swscidxr))
		// ksatC, tiC, gC := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid)
		swsr, celr = make(map[int]float64, len(swscidxr)), make(map[int]float64, len(swscidxr))
		for k, v := range swscidxr {
			ksat := make(map[int]float64)
			var recurs func(int)
			recurs = func(cid int) {
				if gg, ok := b.mpr.isg[cid]; ok {
					if sg, ok := b.mpr.sg[gg]; ok {
						ksat[cid] = sg.Ksat * ts // [m/ts]
						for _, upcid := range b.strc.t.UpIDs(cid) {
							if _, ok := swscidxr[upcid]; !ok { // not including outlet/stream cells
								recurs(upcid)
							}
						}
					} else {
						log.Fatalf(" toDefaultSample.buildTopmodel error, no SurfGeo assigned to type %d", gg)
					}
				} else {
					log.Fatalf(" toDefaultSample.buildTopmodel error, no SurfGeo assigned to cell ID %d", cid)
				}
			}
			recurs(k)

			if len(ksat) != len(v) {
				log.Fatalf(" toDefaultSample.buildTopmodel topology error")
			}
			if b.frc.Q0 <= 0. {
				log.Fatalf(" toDefaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.Q0)
			}

			var gwt gwru.TMQ
			Qo1 := Qo * ts / 365.24 / 86400. // [m/yr] to [m/ts]
			gwt.New(ksat, b.strc.t, b.strc.w, Qo1, m)
			// ti, g := gwt.New(ksat, b.strc.t, b.strc.w, Qo1, m)
			// for i, k := range ksat {
			// 	ksatC[i] = k
			// 	tiC[i] = ti[i]
			// 	gC[i] = g
			// }
			gw[k] = &gwt
			swsr[k] = gwt.Ca / b.contarea // groundwatershed area to catchment area
			celr[k] = gwt.Ca / b.strc.a   // groundwatershed area to cell area
		}
		// saveBinaryMap1(tiC, "tmq.topographic_index.rmap")
		// saveBinaryMap1(gC, "tmq.gamma.rmap")
		// saveBinaryMap1(ksatC, "tmq.ksat_mpts.rmap")
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
		ws:   ws,
		gw:   gw,
		p0:   b.buildSfrac(fcasc),
		swsr: swsr,
		celr: celr,
		// p0: b.buildC0(ns, ts), // ,
		// p1: b.buildC2(ns, ts),
	}
}
