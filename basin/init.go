package basin

import (
	"log"
	"math"
	"sync"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/rdrr/lusg"
)

const (
	secperday = 86400.
	minslope  = 0.001
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

func (b *subdomain) toDefaultSample(m, fcasc float64) sample {
	var wg sync.WaitGroup

	ts := b.frc.h.IntervalSec() // [s/ts]
	if ts <= 0. {
		log.Fatalf(" toDefaultSample error, timestep (IntervalSec) = %v", ts)
	}
	ws := make(hru.WtrShd, b.ncid)
	var gw map[int]*gwru.TMQ
	var swsr, celr map[int]float64

	assignHRUs := func() {
		// defer fmt.Println("  assignHRUs complete")
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
		// defer fmt.Println("  buildTopmodel complete")
		defer wg.Done()
		if b.frc.Q0 <= 0. {
			log.Fatalf(" toDefaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.Q0)
		}

		swscidxr := make(map[int][]int, len(b.rtr.sws)) // id'd by outlet cell (typically a stream cell)
		for k, v := range b.rtr.sws {
			if _, ok := swscidxr[v]; !ok {
				swscidxr[v] = []int{k}
			} else {
				swscidxr[v] = append(swscidxr[v], k)
			}
		}
		nsws := len(swscidxr)

		type kv struct {
			k int
			v gwru.TMQ
		}
		var wg1 sync.WaitGroup
		ch := make(chan kv, nsws)
		getgw := func(sid int) {
			defer wg1.Done()
			ksat := make(map[int]float64)
			for _, c := range swscidxr[sid] {
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
			gwt.New(ksat, b.strc.u, b.strc.t, b.strc.w, m, b.frc.Q0)
			ch <- kv{k: sid, v: gwt}
		}

		for k := range swscidxr {
			if b.cid0 >= 0 {
				eval := make(map[int]bool)
				for _, c := range b.strc.t.UpIDs(b.cid0) {
					if _, ok := eval[b.rtr.sws[c]]; !ok {
						eval[b.rtr.sws[c]] = true
						wg1.Add(1)
						go getgw(k)
					}
				}
			} else {
				log.Fatalf(" toDefaultSample.buildTopmodel: TODO")
			}
		}
		wg1.Wait()
		close(ch)
		gw, swsr, celr = make(map[int]*gwru.TMQ, nsws), make(map[int]float64, nsws), make(map[int]float64, nsws)
		for kv := range ch {
			k, gwt := kv.k, kv.v
			gw[k] = &gwt
			swsr[k] = gwt.Ca / b.contarea // groundwatershed area to catchment area
			celr[k] = gwt.Ca / b.strc.a   // groundwatershed area to cell area
		}
		return
	}

	wg.Add(2)
	go assignHRUs()
	go buildTopmodel()
	wg.Wait()

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
