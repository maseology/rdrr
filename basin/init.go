package basin

import (
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/rdrr/lusg"
)

func (b *subdomain) buildSfrac(fcasc float64) map[int]float64 {
	fc := make(map[int]float64, len(b.cids))
	for _, c := range b.cids {
		s := b.strc.t.TEC[c].S
		if s <= minslope {
			fc[c] = 0.
		} else if s >= fcasc {
			fc[c] = 1.
		} else {
			fc[c] = math.Log(minslope/s) / math.Log(minslope/fcasc) // see: fuzzy_slope.xlsx
		}
	}
	return fc
	// fc := make(map[int]float64, len(b.cids))
	// for _, c := range b.cids {
	// 	fc[c] = math.Min(fcasc*math.Sqrt(b.strc.t.TEC[c].S), 1.)
	// }
	// return fc
}

func (b *subdomain) toDefaultSample(m, fcasc, soildepth float64) sample {
	var wg sync.WaitGroup

	ts := b.frc.h.IntervalSec() // [s/ts]
	if ts <= 0. {
		log.Fatalf(" toDefaultSample error, timestep (IntervalSec) = %v", ts)
	}
	ws := make(hru.WtrShd, b.ncid)
	var gw map[int]*gwru.TMQ

	assignHRUs := func() {
		defer fmt.Println("  assignHRUs complete")
		defer wg.Done()
		build := func(cid int) {
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
			drnsto, srfsto, fimp, _ := lu.GetSOLRIS1(soildepth) //lu.GetDefaultsSOLRIS()
			h.Initialize(drnsto, srfsto, fimp, sg.Ksat, ts)
			ws[cid] = &h
		}

		if b.cid0 >= 0 {
			var recurs func(int)
			recurs = func(cid int) {
				build(cid)
				for _, upcid := range b.strc.t.UpIDs(cid) {
					recurs(upcid)
				}
			}
			recurs(b.cid0)
		} else {
			for _, c := range b.cids {
				build(c)
			}
		}
	}

	buildTopmodel := func() {
		defer fmt.Println("  buildTopmodel complete")
		defer wg.Done()
		if b.frc.Q0 <= 0. {
			log.Fatalf(" toDefaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.Q0)
		}

		type kv struct {
			k int
			v gwru.TMQ
		}
		nsws := len(b.rtr.swscidxr)
		var wg1 sync.WaitGroup
		ch := make(chan kv, nsws)
		getgw := func(sid int) {
			defer wg1.Done()
			ksat := make(map[int]float64)
			for _, c := range b.rtr.swscidxr[sid] {
				if gg, ok := b.mpr.isg[c]; ok {
					if sg, ok := b.mpr.sg[gg]; ok {
						if sg.Ksat <= 0. {
							log.Fatalf(" toDefaultSample.buildTopmodel error: cell %d has an assigned ksat = %v\n", c, sg.Ksat)
						}
						ksat[c] = sg.Ksat * ts // [m/ts]
					} else {
						log.Fatalf(" toDefaultSample.buildTopmodel error, no SurfGeo assigned to type %d", gg)
					}
				} else {
					log.Fatalf(" toDefaultSample.buildTopmodel error, no SurfGeo assigned to cell ID %d", c)
				}
			}
			var gwt gwru.TMQ
			gwt.New(ksat, b.strms, b.strc.t, b.strc.w, m)
			ch <- kv{k: sid, v: gwt}
		}

		if b.cid0 >= 0 {
			uids := b.strc.t.UpIDs(b.cid0)
			for k := range b.rtr.swscidxr {
				eval := make(map[int]bool)
				for _, c := range uids {
					if _, ok := eval[b.rtr.sws[c]]; !ok {
						eval[b.rtr.sws[c]] = true
						wg1.Add(1)
						go getgw(k)
					}
				}
			}
		} else {
			if len(b.rtr.swscidxr) == 1 {
				wg1.Add(1)
				go getgw(-1)
			} else {
				for k := range b.rtr.swscidxr {
					wg1.Add(1)
					go getgw(k)
				}
			}
		}

		wg1.Wait()
		close(ch)
		gw = make(map[int]*gwru.TMQ, nsws)
		for kv := range ch {
			k, gwt := kv.k, kv.v
			gw[k] = &gwt
		}
		return
	}

	wg.Add(2)
	go assignHRUs()
	go buildTopmodel()
	wg.Wait()

	p0 := b.buildSfrac(fcasc)

	finalAdjustments := func() {
		defer wg.Done()
		// set streams to 100% cascade
		for _, g := range gw {
			for c := range g.Qs {
				p0[c] = 1.
			}
		}
	}

	wg.Add(1)
	go finalAdjustments()
	wg.Wait()

	return sample{
		ws: ws,
		gw: gw,
		p0: p0,
	}
}
