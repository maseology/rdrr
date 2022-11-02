package model

import (
	"log"
	"runtime"
	"sync"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/rdrr/lusg"
)

func (b *subdomain) defaultSample(topm, kstrm, mcasc, soildepth, kfact float64) sample {
	var wg sync.WaitGroup

	ts := b.frc.IntervalSec // [s/ts]
	if ts <= 0. {
		log.Fatalf(" defaultSample error, timestep (IntervalSec) = %v", ts)
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
				log.Fatalf(" defaultSample.assignHRUs error, no LandUse assigned to cell ID %d", cid)
			}
			if gg, ok = b.mpr.SGx[cid]; !ok {
				// log.Fatalf(" defaultSample.assignHRUs error, no SurfGeo assigned to cell ID %d", cid)
				log.Printf(" defaultSample.assignHRUs warning, no SurfGeo assigned to cell ID %d", cid)
				gg = 6 // Unknown (variable)
			}
			var lu lusg.LandUse
			ksat := 1.e-5
			// var sg lusg.SurfGeo
			if lu, ok = b.mpr.LU[ll]; !ok {
				log.Fatalf(" defaultSample.assignHRUs error, no LandUse assigned of type %d", ll)
			}
			if ksat, ok = b.mpr.Ksat[gg]; !ok {
				log.Fatalf(" defaultSample.assignHRUs error, no SurfGeo assigned to type %d", gg)
			}

			var h hru.HRU
			drnsto, srfsto, _, _ := lu.Rebuild1(soildepth, b.mpr.Fimp[cid], b.mpr.Ifct[cid])
			h.Initialize(drnsto, srfsto, b.mpr.Fimp[cid], ksat*kfact*ts, 0., 0.)
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
		// 	log.Fatalf(" defaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.q0)
		// }

		type kv struct {
			k int
			v gwru.TMQ
		}
		nsws := len(b.rtr.SwsCidXR)
		ch := make(chan kv, runtime.NumCPU()/2)
		getgw := func(sid int) {
			ksat, grad := make(map[int]float64, len(b.rtr.SwsCidXR[sid])), make(map[int]float64, len(b.rtr.SwsCidXR[sid]))
			for _, c := range b.rtr.SwsCidXR[sid] {
				gg := 6 // default: unknown/variable
				if _, ok := b.mpr.SGx[c]; ok {
					gg = b.mpr.SGx[c]
				}
				if ks, ok := b.mpr.Ksat[gg]; ok {
					if ks <= 0. {
						log.Fatalf(" defaultSample.buildTopmodel error: cell %d has an assigned ksat = %v\n", c, ks)
					}
					ksat[c] = ks * kfact * ts // [m/ts]
				} else {
					log.Fatalf(" defaultSample.buildTopmodel error, no SurfGeo assigned to type %d", gg)
				}
				// } else {
				// 	log.Fatalf(" defaultSample.buildTopmodel error, no SurfGeo assigned to cell ID %d", c)
				// }
				grad[c] = b.strc.TEM.TEC[c].G
			}
			var gwt gwru.TMQ
			gwt.New(ksat, grad, b.mpr.GW[sid].UCA, b.mpr.GW[sid].StrmXR, b.strc.Wcell, m)
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

	cascf := b.buildCascadeFractionFuzzy(kstrm, mcasc)

	finalAdjustments := func() {
		defer wg.Done()
		for _, g := range gw {
			for c := range g.Qs {
				cascf[c] = kstrm
				ws[c].Sdet.Cap = soildepth // in cases where stream cell courses through a flow-resistive cell, ensure movement of water
				ws[c].Sma.Cap = 0.
			}
		}
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

func (b *subdomain) surfgeoSample(kstrm, mcasc, urbDiv, soildepth float64, topm, ksat []float64) sample {
	var wg sync.WaitGroup

	ts := b.frc.IntervalSec // [s/ts]
	if ts <= 0. {
		log.Fatalf(" surfgeoSample error, timestep (IntervalSec) = %v", ts)
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
				log.Fatalf(" surfgeoSample.assignHRUs error, no LandUse assigned to cell ID %d", cid)
			}

			if gg, ok = b.mpr.SGx[cid]; !ok {
				log.Printf(" surfgeoSample.assignHRUs warning, no SurfGeo assigned to cell ID %d", cid)
				gg = 6 // Unknown (variable)
			}
			if gg == -9999 || gg == 0 {
				log.Printf(" surfgeoSample.assignHRUs warning, no SurfGeo assigned to cell ID %d, gg = %d\n", cid, gg)
				gg = 6
			}
			var lu lusg.LandUse
			if lu, ok = b.mpr.LU[ll]; !ok {
				log.Fatalf(" surfgeoSample.assignHRUs error, no LandUse assigned of type %d", ll)
			}

			var h hru.HRU
			drnsto, srfsto, _, _ := lu.Rebuild1(soildepth, b.mpr.Fimp[cid], b.mpr.Ifct[cid])
			h.Initialize(drnsto, srfsto, b.mpr.Fimp[cid], ksat[gg-1]*ts, 0., 0.)
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

	buildTopmodel := func() {
		// defer fmt.Println("  buildTopmodel complete")
		defer wg.Done()
		// if b.frc.q0 <= 0. {
		// 	log.Fatalf(" surfgeoSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.q0)
		// }

		type kv struct {
			k int
			v gwru.TMQ
		}
		ngws := len(b.mpr.GW)
		ch := make(chan kv, runtime.NumCPU()/2)
		getgw := func(gid int, m float64) {
			ksatTS := make(map[int]float64, len(b.mpr.GW[gid].CidXR))
			gradTS := make(map[int]float64, len(b.mpr.GW[gid].CidXR))
			for _, c := range b.mpr.GW[gid].CidXR {
				gg := 6 // default: unknown/variable
				if _, ok := b.mpr.SGx[c]; ok {
					gg = b.mpr.SGx[c]
				}
				if gg == -9999 || gg == 0 {
					gg = 6
				}
				ksatTS[c] = ksat[gg-1] * ts // [m/ts]
				gradTS[c] = b.strc.TEM.TEC[c].G
			}
			var gwt gwru.TMQ
			gwt.New(ksatTS, gradTS, b.mpr.GW[gid].UCA, b.mpr.GW[gid].StrmXR, b.strc.Wcell, m)
			ch <- kv{k: gid, v: gwt}
		}

		// if b.cid0 >= 0 {
		// 	for s := range b.rtr.SwsCidXR {
		// 		go getgw(s)
		// 	}
		// } else {
		// 	if len(b.rtr.SwsCidXR) == 1 {
		// 		go getgw(-1)
		// 	} else {
		// 		for k := range b.rtr.SwsCidXR {
		// 			go getgw(k)
		// 		}
		// 	}
		// }
		for k, v := range topm {
			go getgw(k, v)
		}

		gw = make(map[int]*gwru.TMQ, ngws)
		for i := 0; i < ngws; i++ {
			kv := <-ch
			k, gwt := kv.k, kv.v
			gw[k] = &gwt
		}
		close(ch)
		return
	}

	wg.Add(2)
	go assignHRUs()
	go buildTopmodel()
	wg.Wait()

	cascf := b.buildCascadeFractionFuzzy(kstrm, mcasc)

	func() { // finalAdjustments
		for _, g := range gw {
			for c := range g.Qs {
				cascf[c] = kstrm
			}
		}
		strms := func() map[int]bool {
			strms := make(map[int]bool, b.nstrm)
			for _, g := range b.mpr.GW {
				for _, c := range g.StrmXR {
					strms[c] = true
				}
			}
			return strms
		}()
		for _, c := range b.cids {
			if _, ok := strms[c]; !ok { // skip stream cells
				if ll, ok := b.mpr.LUx[c]; !ok {
					log.Fatalf(" surfgeoSample finalAdjustments error, no LandUse assigned to cell ID %d", c)
				} else if ll == 4 { // urban
					cascf[c] = urbDiv
				}
			}
		}
	}()

	return sample{
		ws:    ws,
		gw:    gw,
		cascf: cascf,
	}
}
