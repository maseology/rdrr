package model

import (
	"log"
	"math"
	"runtime"
	"sync"

	"github.com/maseology/glbopt"
	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/mmaths"
	"github.com/maseology/rdrr/lusg"
)

func (b *subdomain) buildCascadeFraction(rng float64) map[int]float64 {
	fc := make(map[int]float64, len(b.cids))
	for _, c := range b.cids {
		h := math.Pow(b.strc.TEM.TEC[c].G, 2)
		r := math.Pow(rng, 2)
		fc[c] = (sill-nugget)*(1.-math.Exp(-h/r/a)) + nugget // Gaussian variogram model
	}
	return fc
	// fc := make(map[int]float64, len(b.cids))
	// for _, c := range b.cids {
	// 	s := b.strc.TEM.TEC[c].G
	// 	if s <= minslope {
	// 		fc[c] = 0.
	// 	} else if s >= smax {
	// 		fc[c] = 1.
	// 	} else {
	// 		fc[c] = math.Log(minslope/s) / math.Log(minslope/smax) // see: fuzzy_slope.xlsx
	// 	}
	// }
	// return fc
}

func (b *subdomain) toDefaultSample(topm, slpx, soildepth, kfact float64) sample {
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

	cascf := b.buildCascadeFraction(slpx)

	finalAdjustments := func() {
		defer wg.Done()
		for _, g := range gw {
			for c := range g.Qs {
				cascf[c] = 1.              // set streams to 100% cascade
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

func (pp *evaluation) initialize(Dinc, m float64, print bool) {
	smpl := func(u float64) float64 {
		return mmaths.LinearTransform(-10., 10., u)
	}
	opt := func(u []float64) float64 {
		hb := 0.
		dm := smpl(u[0])
		for i, v := range pp.strmQs {
			hb += v * math.Exp((Dinc-dm-pp.drel[i])/m)
		}
		hb /= pp.fncid
		return math.Abs(hb-avgRch) / avgRch
	}
	u, _ := glbopt.Fibonacci(opt)
	pp.dm = smpl(u)

	pp.s0s = 0.
	for i := 0; i < int(pp.fncid); i++ {
		pp.s0s += pp.ws[i].Storage() // initial subsample storage
	}
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

// func (pp *evaluation) initialize(Dinc, m float64, print bool) {

// 	g := 0.
// 	for _, v := range pp.strmQs {
// 		g += v * math.Exp((Dinc-d)/m)
// 	}
// 	g /= float64(len(pp.strmQs))
// 	fmt.Println(len(pp.strmQs), g, avgRch, pp.fncid, math.Log(avgRch*pp.fncid))
// 	pp.dm = -m * (g + math.Log(avgRch*pp.fncid))

// 	// pp.dm = func() (dm float64) {
// 	// 	dm = -1. //0. //-m * math.Log(q0/Qs) // q0 = avgRch // default discharge for warm-up
// 	// 	if len(pp.strmQs) == 0 {
// 	// 		return
// 	// 	}
// 	// 	q0t, n := 0., 0
// 	// 	for {
// 	// 		for i, v := range pp.strmQs {
// 	// 			q0t += v * math.Exp((Dinc-dm-pp.drel[i])/m)
// 	// 		}
// 	// 		q0t /= pp.fncid
// 	// 		if q0t <= avgRch {
// 	// 			if print && dm <= 0. {
// 	// 				t := math.Abs(math.Log10(avgRch / q0t))
// 	// 				if t < 1.33 {
// 	// 					fmt.Printf("  evaluation.initialize: steady reached without iterations -- rch %.2e; Qo %.2e\n", avgRch, q0t)
// 	// 				} else {
// 	// 					fmt.Printf("  evaluation.initialize: initial discharge imposed without iteration -- rch %.2e; Qo %.2e\n", avgRch, q0t)
// 	// 				}
// 	// 			}
// 	// 			break
// 	// 		}
// 	// 		dm += .1
// 	// 		q0t = 0.
// 	// 		n++
// 	// 		if n > steadyiter {
// 	// 			if print {
// 	// 				fmt.Println("  evaluation.initialize: steady reached max iterations")
// 	// 			}
// 	// 			break
// 	// 		}
// 	// 	}
// 	// 	return
// 	// }()
// 	// pp.dm = 4.
// 	pp.s0s = 0.
// 	for i := 0; i < int(pp.fncid); i++ {
// 		pp.s0s += pp.ws[i].Storage() // initial subsample storage
// 	}
// }
