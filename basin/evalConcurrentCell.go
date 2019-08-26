package basin

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/maseology/goHydro/hru"
	"github.com/maseology/goHydro/met"
	"github.com/maseology/mmio"
	"github.com/maseology/objfunc"
)

const (
	maxlag = 1.0 // [m] maximum allowable depth of water on a cell
)

type node struct {
	i     int
	ws    *hru.HRU
	us    []*node
	ds    *node
	fcasc float64
	pg    chan kv
}

type kv struct {
	k int
	v float64
}

func (p *sample) newDirectedGraph(downstream map[int]int) (map[int]*node, map[int]bool) {
	nodes := make(map[int]*node, len(downstream))
	roots := make(map[int]bool, 1)
	for k := range downstream {
		nodes[k] = &node{i: k, pg: make(chan kv), ws: p.ws[k], fcasc: p.p0[k]}
	}
	for k, v := range downstream {
		nodes[k].ds = nodes[v]
	}
	for _, v := range nodes {
		if v.ds != nil {
			v.ds.us = append(v.ds.us, v)
		} else {
			roots[v.i] = true
		}
	}
	return nodes, roots
}

type output struct {
	i                      int
	a, r, g, lag, ds, wbal float64
}

func (n *node) eval(wg *sync.WaitGroup, y, di, ep float64, out chan<- output) {
	wg.Add(1)
	defer wg.Done()

	coll, cnt, brk := make(map[int]float64, len(n.us)), 0, false
	for {
		select {
		case kv := <-n.pg:
			coll[kv.k] = kv.v
			cnt++
		default:
			brk = cnt == len(n.us)
		}
		if brk {
			// fmt.Println(n.i, len(n.us))
			break
		}
	}

	// released, run hydrology:
	ro := 0. // runon
	for _, u := range n.us {
		if v, ok := coll[u.i]; ok {
			ro += v
		} else {
			log.Fatalf(" upslope cell %d not complete prior to %d\n", u.i, n.i)
		}
	}

	// update HRU
	s0 := n.ws.Storage()              // initial storage
	rgen := n.ws.UpdateP(y + ro - di) // (generated) runoff
	g := 0.                           // recharge
	if di >= 0. {                     // only recharge when deficit is available; otherwise reject
		g = n.ws.UpdatePerc()
	}
	a := n.ws.UpdateEp(ep) // aet
	if a < 0. {
		log.Fatalf(" hru water-balance error (cell %d), HRU ET = %.3e mm\n", n.i, a*1000.)
	}
	if rgen < 0. {
		log.Fatalf(" hru water-balance error (cell %d), HRU runoff = %.3e mm\n", n.i, rgen*1000.)
	}
	if g < 0. {
		log.Fatalf(" hru water-balance error (cell %d), HRU potential recharge = %.3e mm\n", n.i, g*1000.)
	}
	ds := n.ws.Storage() - s0 // change in storage

	r := rgen * n.fcasc
	lag := rgen * (1. - n.fcasc) // retention
	if lag > maxlag {
		r += lag - maxlag
		lag = maxlag
	}

	// water-balance
	wbal := y + ro - di - (a + r + lag + g + ds)
	if math.Abs(wbal) > nearzero {
		fmt.Printf(" cell ID: %d   del-sto:  %.6f\n", n.i, ds)
		fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f   s0: %.5f  aet: %.5f  rch: % .5f  ron: %.5f  roff: %.5f  lag: %.5f\n", y, -di, n.ws.Storage(), s0, a, g, ro, r, lag)
		log.Fatalf(" cell %d: water-balance error, |wbal| = %.5e m\n", n.i, math.Abs(wbal))
	}

	// ping downstream node
	if n.ds != nil {
		n.ds.pg <- kv{k: n.i, v: r}
	} else {
		r = rgen // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case p.pa[c]=0.
		lag = 0.
	}
	out <- output{i: n.i, a: a, r: r, g: g, lag: lag, ds: ds, wbal: wbal}
}

func (b *subdomain) evalConcurrentCell(p *sample, freeboard float64, print bool) (of float64) {
	runtime.GOMAXPROCS(b.ncid)
	var wg sync.WaitGroup
	nds, rts := p.newDirectedGraph(b.ds)
	nnds := len(nds)
	nstep, dtb, dte, intvl := b.frc.trimFrc(15)
	h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor

	// monitors
	o, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0 // outlet discharge [m³/s]: observed, simulated

	// closure
	defer func() {
		fo, fs := mmio.InterfaceToFloat(o), mmio.InterfaceToFloat(s)
		rmse := objfunc.RMSE(fo, fs)
		of = rmse //(1. - kge) //* (1. - mwr2)
		if print {
			kge := objfunc.KGE(fo, fs)
			mwr2 := objfunc.Krause(computeMonthly(dt, fo, fs, float64(intvl), b.contarea))
			nse := objfunc.NSE(fo, fs)
			bias := objfunc.Bias(fo, fs)
			// // sumHydrograph(dt, o, s, bf)
			// sumHydrographWB(dt, ws, wd, wa, wg, wx, wk)
			// sumMonthly(dt, o, s, float64(intvl), b.contarea)
			// saveBinaryMap1(gp, "precipitation.rmap")
			// saveBinaryMap1(ga, "aet.rmap")
			// saveBinaryMap1(gr, "runoff.rmap")
			// saveBinaryMap1(gg, "recharge.rmap")
			// saveBinaryMap1(gl, "mobile.rmap")
			mmio.ObsSim("hyd.png", fo, fs)
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			fmt.Printf("  KGE: %.3f  NSE: %.3f  mon-wr2: %.3f  Bias: %.3f\n", kge, nse, mwr2, bias)
		}
	}()

	laglast := 0.
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		fmt.Println(d)
		frc := b.frc.c[d]

		ysum, xsum := 0., 0.
		ggwsum, ggwcnt, gwdlast := make(map[int]float64, len(p.gw)), make(map[int]float64, len(p.gw)), 0.
		for k, v := range p.gw {
			gwdlast += v.Dm * p.swsr[k] // basin groundwater deficit at beginning of timestep
			ggwsum[k] = 0.              // sum of recharge for gw res k
			ggwcnt[k] = 0.              // count of recharge for gw res k
		}

		out := make(chan output, nnds)
		for _, cid := range b.cids {
			n := nds[cid]
			sid := b.rtr.sws[cid] // subwatershed id

			y := frc[met.AtmosphericYield]                                  // precipitation/atmospheric yield (rainfall + snowmelt)
			ep := frc[met.AtmosphericDemand] * b.strc.f[cid][d.YearDay()-1] // evaporative demand, adjusted for slope-aspect
			di := p.gw[sid].GetDi(cid)                                      // groundwater deficit
			if di < -freeboard {                                            // saturation excess runoff/groundwater discharge (di: groundwater deficit)
				di += freeboard
				xsum -= di
				ggwsum[sid] += di
			} else {
				di = 0.
			}

			go n.eval(&wg, y, di, ep, out)
			ysum += y
		}
		fmt.Printf("%d ", runtime.NumGoroutine()-1)
		wg.Wait()
		close(out)

		// water balance
		wbsum, asum, rsum, gsum, lagsum, dssum := 0., 0., 0., 0., 0., 0.
		for v := range out {
			sid := b.rtr.sws[v.i]
			ggwsum[sid] += v.g
			ggwcnt[sid]++

			// water balance
			wbsum += v.wbal
			asum += v.a
			if _, ok := rts[v.i]; ok {
				rsum += v.r // only collecting runoff from outlets
			}
			gsum += v.g
			lagsum += v.lag
			dssum += v.ds
		}
		wbsum /= b.fncid
		ysum /= b.fncid
		asum /= b.fncid
		rsum /= b.fncid
		gsum /= b.fncid
		lagsum /= b.fncid
		dssum /= b.fncid
		xsum /= b.fncid

		if math.Abs(wbsum) > nearzero {
			fmt.Printf(" step: %d  freeboard: %.5f\n", i, freeboard)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, lagsum, asum, gsum, rsum, frc[met.UnitDischarge])
			log.Fatalf(" (integrated) hru water-balance error, |wbsum| = %.5e m\n", math.Abs(wbsum))
		}

		gwd := 0.
		for k, v := range p.gw {
			gwd += v.Dm * p.swsr[k] // basin groundwater deficit at end of timestep
		}
		wbalBasin := ysum + xsum - gwdlast + laglast - (-gwd + asum + rsum + gsum + lagsum + dssum)
		if math.Abs(wbalBasin) > nearzero {
			fmt.Printf(" step: %d  freeboard: %.5f\n", i, freeboard)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, lagsum, asum, gsum, rsum, frc[met.UnitDischarge])
			fmt.Printf(" del-sto: %.5f  del-gw: %.5f  wbalBasin: % .10f\n", dssum, gwd-gwdlast, wbalBasin)
			log.Fatalf(" basin water-balance error, |wbalBasin| = %.5e m\n", math.Abs(wbalBasin))
		}

		// save results
		dt[i] = d
		o[i] = frc[met.UnitDischarge] * h2cms
		s[i] = rsum * h2cms
		// ws[i] = ssum * 1000. // CE storage
		// wd[i] = gwd * 1000.  // GW deficit
		// wg[i] = gsum * 1000. // groundwater recharge
		// wx[i] = xsum * 1000. // saturation excess runoff
		// wk[i] = slag * 1000. // mobile runoff
		// wa[i] = asum * 1000. // evaporation
		fmt.Println(runtime.NumGoroutine())
		i++
		// laglast = lagsum
	}
	return
}
