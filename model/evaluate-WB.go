package model

import (
	"fmt"
	"log"
	"math"
)

// evalWB is the main model routine, the others are derivatives to this. m: TOPMODEL parameter
func evalWB(p *evaluation, res resulter, monid []int) {
	ncid := int(p.fncid)
	obsCms, fcms := make(map[int]monitor, len(monid)), p.ca/p.intvl
	sim, hsto, gsto := make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep)
	// yss, ass, rss, gss, bss := 0., 0., 0., 0., 0.
	// distributed monitors [mm/yr]
	gy, ge, ga, gr, gg, gb := make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid)
	// temporal sws monitors [m/ts]
	ty, ti, ta, to, tg, ts, tb, tdm := make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep)

	for _, c := range monid {
		obsCms[p.cxr[c]] = monitor{c: c, v: make([]float64, p.nstep)}
	}

	dm, dm0, s0s := p.dm, p.dm, p.s0s // initial condition
	for k := 0; k < p.nstep; k++ {
		// doy := p.t[k].doy // day of year
		// if k%100 == 0 {
		// 	fmt.Printf("%.3f ", dm)
		// }

		ys, ins, as, outs, gs, s1s, bs := 0., 0., 0., 0., 0., 0., 0.
		for i, v := range p.sources {
			p.ws[i].Sdet.Sto += v[k] // inflow from up sws
			ins += v[k]
		}
		for i := 0; i < ncid; i++ {
			s0 := p.ws[i].Storage()
			y := p.y[p.mxr[i]][k]
			ep := p.ep[p.mxr[i]][k]               // p.f[i][doy] // p.ep[k][0] // p.f[i][doy] // p.ep[k][0] * p.f[i][doy]
			d := (dm + p.drel[i]) / p.m[p.gxr[i]] // groundwater deficit (relative to topmodel m)
			cascf := p.cascf[i]
			a, r, g := p.ws[i].UpdateWT(y, ep, d < 0.)

			p.ws[i].Sdet.Sto += r * (1. - cascf)
			r *= cascf
			g += p.ws[i].InfiltrateSurplus() // help to maintain "water towers"
			s1 := p.ws[i].Storage()
			s1s += s1

			b := 0.
			if v, ok := p.strmQs[i]; ok { // stream cells always cascade
				b = v * math.Exp(-d)
				bs += b
				gb[i] += b
				r += b
			}

			// water balance
			hruwbal := y + s0 + b - (a + r + g + s1)
			if math.Abs(hruwbal) > nearzero {
				// fmt.Printf("|hruwbal| = %e\n", hruwbal)
				fmt.Print("^")
			}

			ys += y
			as += a
			gy[i] += y
			ge[i] += ep
			ga[i] += a

			if _, ok := obsCms[i]; ok {
				obsCms[i].v[k] = r * fcms
			}
			if p.ds[i] == -1 { // outlet cell
				outs += r
			} else {
				p.ws[p.cxr[p.ds[i]]].Sdet.Sto += r
			}
			gs += g
			gr[i] += r
			gg[i] += g
		}

		// update GWR
		dm += (bs - gs) / p.fncid

		// basin sums
		// yss += ys
		// ass += as
		// rss += rs
		// gss += gs
		// bss += bs
		// bf[k] = bs

		sim[k] = outs
		hsto[k] = s1s / p.fncid
		gsto[k] = dm

		ty[k] = ys / p.fncid
		ti[k] = ins / p.fncid
		ta[k] = as / p.fncid
		to[k] = outs / p.fncid
		tg[k] = gs / p.fncid
		tb[k] = bs / p.fncid
		ts[k] = s1s / p.fncid
		tdm[k] = dm

		// water balances
		allhruwbal := ys + ins + bs + s0s - (as + outs + gs + s1s)
		if math.Abs(allhruwbal) > nearzero {
			// fmt.Printf("sum{hruwbal} = %e\n", allhruwbal)
			fmt.Print("*")
		}

		basinwbal := ys + ins + (dm-dm0)*p.fncid + s0s - (as + outs + s1s)
		if math.Abs(basinwbal) > nearzero {
			if math.Abs(basinwbal) > fatalzero {
				log.Fatalf("waterbalance error |basinwbal| = %e, step %d", basinwbal, k)
			}
			// fmt.Printf("basinwbal = %e\n", basinwbal)
			fmt.Print("+")
		}

		gwbal := (dm-dm0)*p.fncid + gs - bs
		if math.Abs(gwbal) > nearzero {
			if math.Abs(gwbal) > fatalzero {
				log.Fatalf("waterbalance error |gwbal| = %e, step %d", gwbal, k)
			}
			// fmt.Printf("|gwbal| = %e\n", gwbal)
			fmt.Print("~")
		}

		// save state
		s0s = s1s
		dm0 = dm
	}

	for k := 1; k < p.nstep; k++ {
		twbal := (ty[k] + ti[k] + ts[k-1] - tdm[k-1]) - (ta[k] + to[k] + ts[k] - tdm[k])
		if math.Abs(twbal) > nearzero {
			if math.Abs(twbal) > fatalzero {
				s1 := fmt.Sprintf("    inputs: atmyld (y=%.5f); inflow (i=%.5f); gwd (b=%.5f); TOTAL = %.5f\n", ty[k], ti[k], tb[k], ty[k]+ti[k]+tb[k])
				s1 += fmt.Sprintf("   outputs:    aet (e=%.5f); outflw (o=%.5f); rch (g=%.5f); TOTAL = %.5f\n", ta[k], to[k], tg[k], ta[k]+to[k]+tg[k])
				s1 += fmt.Sprintf("     gwdef:     dm(t)=%10.5f     dm(t-1)=%10.5f\t\tDIFF = %10.5f\n", tdm[k], tdm[k-1], tdm[k]-tdm[k-1])
				s1 += fmt.Sprintf("   storage:      s(t)=%10.5f      s(t-1)=%10.5f\t\tDIFF = %10.5f\n", ts[k-1], ts[k], ts[k-1]-ts[k])
				log.Fatalf("waterbalance error |twbal| = %e, step %d\n%s", twbal, k, s1)
			}
			// fmt.Printf("|twbal| = %e\n", twbal)
			fmt.Print("~")
		}
	}

	func() {
		res.getTotals(sim, hsto, gsto)
		for _, v := range obsCms {
			gwg.Add(1)
			go v.print(p.dir)
		}
		g := gmonitor{gy, ge, ga, gr, gg, gb, p.dir}
		gwg.Add(1)
		go g.print(p.ws, p.sources, p.cxr, p.ds, p.intvl, float64(p.nstep))
		tm := tmonitor{p.sid, ty, ti, ta, to, tg, ts, tb, tdm, p.dir}
		gwg.Add(1)
		go tm.print(p.s0s, p.dm)
	}()
	return
}
