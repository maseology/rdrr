package model

import (
	"log"
	"runtime"
	"sync"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/rdrr/lusg"
)

func (b *subdomain) toDefaultSample(topm, grdMin, kstrm, mcasc, soildepth, kfact float64) sample {
	var wg sync.WaitGroup

	ts := b.frc.IntervalSec // [s/ts]
	if ts <= 0. {
		log.Fatalf(" toDefaultSample error, timestep (IntervalSec) = %v", ts)
	}
	ws := make(hru.WtrShd, b.ncid)
	var gw map[int]*gwru.TMQ

	assignHRUs := func() {
		// defer fmt.Println("  assignHRUs complete")
		defer wg.Done()
		build := func(cid int) {
			var ll, gg int
			var ok bool
			if ll, ok = b.mpr.LUx[cid]; !ok {
				log.Fatalf(" toDefaultSample.assignHRUs error, no LandUse assigned to cell ID %d", cid)
			}
			if gg, ok = b.mpr.SGx[cid]; !ok {
				// log.Fatalf(" toDefaultSample.assignHRUs error, no SurfGeo assigned to cell ID %d", cid)
				log.Printf(" toDefaultSample.assignHRUs warning, no SurfGeo assigned to cell ID %d", cid)
				gg = 6 // Unknown (variable)
			}
			var lu lusg.LandUse
			var sg lusg.SurfGeo
			if lu, ok = b.mpr.LU[ll]; !ok {
				log.Fatalf(" toDefaultSample.assignHRUs error, no LandUse assigned of type %d", ll)
			}
			if sg, ok = b.mpr.SG[gg]; !ok {
				log.Fatalf(" toDefaultSample.assignHRUs error, no SurfGeo assigned to type %d", gg)
			}

			var h hru.HRU
			drnsto, srfsto, _, _ := lu.Rebuild1(soildepth, b.mpr.Fimp[cid], b.mpr.Ifct[cid])
			h.Initialize(drnsto, srfsto, b.mpr.Fimp[cid], sg.Ksat*kfact*ts, 0., 0.)
			ws[cid] = &h
		}

		if b.cid0 >= 0 {
			var recurs func(int)
			recurs = func(cid int) {
				build(cid)
				for _, upcid := range b.strc.TEM.USlp[cid] {
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

	buildTopmodel := func(m float64) {
		// defer fmt.Println("  buildTopmodel complete")
		defer wg.Done()
		// if b.frc.q0 <= 0. {
		// 	log.Fatalf(" toDefaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.q0)
		// }

		type kv struct {
			k int
			v gwru.TMQ
		}
		nsws := len(b.rtr.SwsCidXR)
		ch := make(chan kv, runtime.NumCPU()/2)
		getgw := func(sid int) {
			ksat := make(map[int]float64)
			for _, c := range b.rtr.SwsCidXR[sid] {
				gg := 6 // default: unknown/variable
				if _, ok := b.mpr.SGx[c]; ok {
					gg = b.mpr.SGx[c]
				}
				if sg, ok := b.mpr.SG[gg]; ok {
					if sg.Ksat <= 0. {
						log.Fatalf(" toDefaultSample.buildTopmodel error: cell %d has an assigned ksat = %v\n", c, sg.Ksat)
					}
					ksat[c] = sg.Ksat * kfact * ts // [m/ts]
				} else {
					log.Fatalf(" toDefaultSample.buildTopmodel error, no SurfGeo assigned to type %d", gg)
				}
				// } else {
				// 	log.Fatalf(" toDefaultSample.buildTopmodel error, no SurfGeo assigned to cell ID %d", c)
				// }
			}
			var gwt gwru.TMQ
			gwt.New(ksat, b.rtr.UCA[sid], b.rtr.SwsStrmXR[sid], b.strc.TEM, b.strc.Wcell, m)
			ch <- kv{k: sid, v: gwt}
		}

		if b.cid0 >= 0 {
			for s := range b.rtr.SwsCidXR {
				go getgw(s)
			}
		} else {
			if len(b.rtr.SwsCidXR) == 1 {
				go getgw(-1)
			} else {
				for k := range b.rtr.SwsCidXR {
					go getgw(k)
				}
			}
		}

		gw = make(map[int]*gwru.TMQ, nsws)
		for i := 0; i < nsws; i++ {
			kv := <-ch
			k, gwt := kv.k, kv.v
			gw[k] = &gwt
		}
		close(ch)
		return
	}

	wg.Add(2)
	go assignHRUs()
	go buildTopmodel(topm)
	wg.Wait()

	cascf := b.buildCascadeFraction(grdMin, kstrm, mcasc)

	finalAdjustments := func() {
		defer wg.Done()
		for _, g := range gw {
			for c := range g.Qs {
				cascf[c] = kstrm
				ws[c].Sdet.Cap = soildepth // in cases where stream cell courses through a flow-resistive cell, ensure movement of water
				ws[c].Sma.Cap = 0.
			}
			// minDrel := math.MaxFloat64
			// for _, v := range g.D {
			// 	if v < minDrel {
			// 		minDrel = v
			// 	}
			// }
			// for c := range g.D {
			// 	if _, ok := b.mpr.LKx[c]; ok {
			// 		g.D[c] = minDrel // presume lakes relative deficits to be equivalent to the SWS min
			// 	}
			// }
		}
		// for c := range b.mpr.LKx {
		// 	fmt.Println(c)
		// 	cascf[c] = 1. // set open water to 100% cascade
		// }
	}

	wg.Add(1)
	go finalAdjustments()
	wg.Wait()

	return sample{
		ws:    ws,
		gw:    gw,
		cascf: cascf,
	}
}