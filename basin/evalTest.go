package basin

import (
	"fmt"
	"math"
	"time"

	"github.com/maseology/goHydro/hru"
	"github.com/maseology/goHydro/met"

	"github.com/maseology/mmio"
	"github.com/maseology/objfunc"
)

const (
	nearzero   = 1e-8
	steadyiter = 500
)

func dehash(b *subdomain, p *sample) (xr map[int]int, strm map[int]float64, frc [][]float64, ws []hru.HRU, drel, p0 []float64, ds []int, nstep, intvl int) {
	ns, dtb, dte, itvl := b.frc.trimFrc(-1)
	nstep, intvl = ns, int(itvl)
	frc = make([][]float64, ns)
	k := 0
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		v := b.frc.c[d]
		frc[k] = []float64{v[met.AtmosphericYield], v[met.AtmosphericDemand], v[met.UnitDischarge]}
		k++
	}
	drel = make([]float64, b.ncid) // initialize mean TOPMODEL deficit
	ws, p0, ds = make([]hru.HRU, b.ncid), make([]float64, b.ncid), make([]int, b.ncid)
	xr = make(map[int]int, b.ncid) // cellID to slice id cross-reference
	strm = make(map[int]float64, b.nstrm)
	for i, c := range b.cids {
		sid := b.rtr.sws[c] // groundwatershed id
		drel[i] = p.gw[sid].D[c]
		ws[i] = *p.ws[c]
		p0[i] = 1. //p.p0[c]
		ds[i] = b.ds[c]
		xr[c] = i
		if v, ok := p.gw[sid].Qs[c]; ok {
			strm[i] = v
		}
	}
	return
}

func report(o, s []float64, ytot, atot, rtot, gtot, fncid float64, nstep int, print bool) (of float64) {
	rmse := objfunc.RMSE(o[365:], s[365:])
	of = rmse
	if print {
		kge := objfunc.KGE(o, s)
		// mwr2 := objfunc.Krause(computeMonthly(dt[365:], o[365:], s[365:], float64(intvl), b.contarea))
		nse := objfunc.NSE(o, s)
		bias := objfunc.Bias(o, s)
		ff := 365.24 * 1000. / float64(nstep) / fncid
		fmt.Printf("  waterbudget [mm/yr]: pre: %.0f  aet: %.0f  rch: %.0f  ro: %.0f  dif: %.1f\n", ytot*ff, atot*ff, gtot*ff, rtot*ff, (ytot-(atot+gtot+rtot))*ff)
		fmt.Printf("  KGE: %.3f  NSE: %.3f  RMSE: %.3f  mon-wR²: NA  Bias: %.3f\n", kge, nse, rmse, bias)
	}
	return
}

// evalTest 6: with topology, topmodel gwr and gw feedback (only 1 gw reservoir)
func (b *subdomain) evalTest(p *sample, Ds, m float64, print bool) (of float64) {
	xr, strm, frc, ws, drel, p0, ds, nstep, intvl := dehash(b, p)
	h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor
	obs, sim, bf := make([]float64, nstep), make([]float64, nstep), make([]float64, nstep)
	yss, ass, rss, gss, bss := 0., 0., 0., 0., 0.
	defer func() {
		mmio.ObsSim("hyd.png", obs, sim, bf, nil)
		of = report(obs, sim, yss, ass, rss, gss, b.fncid, nstep, print)
	}()

	dm := func() (dm float64) {
		q0t, q0, n := 0., b.frc.Q0, 0
		dm = 0. //-m * math.Log(q0/Qs)
		for {
			for c, v := range strm {
				q0t += v * math.Exp((Ds-dm-drel[xr[c]])/m)
			}
			q0t /= b.fncid
			if q0t <= q0 {
				break
			}
			dm += .1
			q0t = 0.
			n++
			if n > steadyiter {
				fmt.Println("steady reached max iterations")
				break
			}
		}
		return
	}()

	for k := 0; k < nstep; k++ {
		v := frc[k]
		obs[k] = v[2] * h2cms // met.UnitDischarge

		ys, as, rs, gs, s0s, s1s, bs, dm0 := 0., 0., 0., 0., 0., 0., 0., dm
		for i := range b.cids {
			s0s += ws[i].Storage()
		}
		for i := range b.cids {
			y := v[0]  // v[met.AtmosphericYield]   // precipitation/atmospheric yield (rainfall + snowmelt)
			ep := v[1] // v[met.AtmosphericDemand] // evaporative demand

			s0 := ws[i].Storage()
			a, r, g := ws[i].UpdateWT(y, ep, dm+drel[i])
			ws[i].AddToStorage(r * (1. - p0[i]))
			r *= p0[i]
			s1 := ws[i].Storage()
			s1s += s1

			hruwbal := y + s0 - (a + r + g + s1)
			if math.Abs(hruwbal) > nearzero {
				fmt.Printf("|hruwbal| = %e\n", hruwbal)
			}

			ys += y
			as += a
			if v, ok := strm[i]; ok {
				hb := v * math.Exp((Ds-dm-drel[i])/m)
				bs += hb
				r += hb
			}
			if ds[i] == -1 { // outlet cell
				rs += r
			} else {
				ws[xr[ds[i]]].AddToStorage(r)
			}
			gs += g
		}
		yss += ys
		ass += as
		rss += rs
		gss += gs
		bss += bs
		sim[k] = rs / b.fncid * h2cms
		bf[k] = bs / b.fnstrm / b.fncid * h2cms
		dm += (bs - gs) / b.fncid

		hruwbal := ys + bs + s0s - (as + rs + gs + s1s)
		if math.Abs(hruwbal) > nearzero {
			fmt.Printf("(sum)|hruwbal| = %e\n", hruwbal)
		}

		basinwbal := ys + (dm-dm0)*b.fncid + s0s - (as + rs + s1s)
		// basinwbal := (dm - dm0) + (gs-bs)/b.fncid // gwbal
		// basinwbal := (dm - dm0) + gs/b.fncid - bs/b.fnstrm
		if math.Abs(basinwbal) > nearzero {
			fmt.Printf("|basinwbal| = %e\n", basinwbal)
		}
	}
	return
}

// // evalTest 5: with topology, topmodel gwr and gw feedback (only 1 gw reservoir)
// func (b *subdomain) evalTest(p *sample, Ds, m float64, print bool) (of float64) {
// 	xr, strm, frc, ws, drel, p0, ds, nstep, intvl := dehash(b, p)
// 	h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor
// 	obs, sim, bf := make([]float64, nstep), make([]float64, nstep), make([]float64, nstep)
// 	yss, ass, rss, gss, bss := 0., 0., 0., 0., 0.
// 	defer func() {
// 		mmio.ObsSim("hyd.png", obs, sim, bf, nil)
// 		of = report(obs, sim, yss, ass, rss, gss, b.fncid, nstep, print)
// 	}()

// 	dm := func() (dm float64) {
// 		q0t, q0, n := 0., b.frc.Q0, 0
// 		dm = 0. //-m * math.Log(q0/Qs)
// 		for {
// 			for c, v := range strm {
// 				q0t += v * math.Exp((Ds-dm-drel[xr[c]])/m)
// 			}
// 			q0t /= b.fnstrm
// 			if q0t <= q0 {
// 				break
// 			}
// 			dm += .1
// 			q0t = 0.
// 			n++
// 			if n > 100 {
// 				fmt.Println("steady reached max iterations")
// 				break
// 			}
// 		}
// 		return
// 	}()

// 	for k := 0; k < nstep; k++ {
// 		v := frc[k]
// 		obs[k] = v[2] * h2cms // met.UnitDischarge

// 		ys, as, rs, gs, s0s, s1s, bs := 0., 0., 0., 0., 0., 0., 0.
// 		for i := range b.cids {
// 			s0s += ws[i].Storage()
// 		}
// 		for i := range b.cids {
// 			y := v[0]  // v[met.AtmosphericYield]   // precipitation/atmospheric yield (rainfall + snowmelt)
// 			ep := v[1] // v[met.AtmosphericDemand] // evaporative demand

// 			s0 := ws[i].Storage()
// 			a, r, g := ws[i].UpdateWT(y, ep, dm+drel[i])
// 			ws[i].AddToStorage(r * (1. - p0[i]))
// 			r *= p0[i]
// 			s1 := ws[i].Storage()
// 			s1s += s1

// 			hruwbal := y + s0 - (a + r + g + s1)
// 			if math.Abs(hruwbal) > nearzero {
// 				fmt.Printf("|hruwbal| = %e\n", hruwbal)
// 			}

// 			ys += y
// 			as += a
// 			if v, ok := strm[i]; ok {
// 				hb := v * math.Exp((Ds-dm-drel[i])/m)
// 				bs += hb
// 				r += hb
// 			}
// 			if ds[i] == -1 { // outlet cell
// 				rs += r
// 			} else {
// 				ws[xr[ds[i]]].AddToStorage(r)
// 			}
// 			gs += g
// 		}
// 		yss += ys
// 		ass += as
// 		rss += rs
// 		gss += gs
// 		bss += bs
// 		sim[k] = (rs/b.fncid + bs/b.fnstrm) * h2cms
// 		bf[k] = bs / b.fnstrm * h2cms
// 		dm += bs/b.fnstrm - gs/b.fncid

// 		hruwbal := ys + bs + s0s - (as + rs + gs + s1s)
// 		if math.Abs(hruwbal) > nearzero {
// 			fmt.Printf("(sum)|hruwbal| = %e\n", hruwbal)
// 		}
// 	}
// 	return
// }

// // evalTest 4: with topology, topmodel gwr and gw feedback (only 1 gw reservoir)
// func (b *subdomain) evalTest(p *sample, Qs, m float64, print bool) (of float64) {
// 	nstep, dtb, dte, intvl := b.frc.trimFrc(-1)
// 	h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor
// 	obs, sim, bf, k := make([]float64, nstep), make([]float64, nstep), make([]float64, nstep), 0
// 	yss, ass, rss, gss := 0., 0., 0., 0.
// 	defer func() {
// 		mmio.ObsSim("hyd.png", obs, sim, bf, nil)
// 		of = report(obs, sim, yss, ass, rss, gss, b.fncid, nstep, print)
// 	}()

// 	dm, drel := -m*math.Log(b.frc.Q0/Qs), make([]float64, b.ncid) // initialize mean TOPMODEL deficit
// 	ws, p0, ds := make([]hru.HRU, b.ncid), make([]float64, b.ncid), make([]int, b.ncid)
// 	xr := make(map[int]int, b.ncid)
// 	for i, c := range b.cids {
// 		sid := b.rtr.sws[c] // groundwatershed id
// 		drel[i] = p.gw[sid].D[c]
// 		ws[i] = *p.ws[c]
// 		p0[i] = p.p0[c]
// 		ds[i] = b.ds[c]
// 		xr[c] = i
// 	}

// 	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
// 		// fmt.Println(d)
// 		v := b.frc.c[d]
// 		obs[k] = v[met.UnitDischarge] * h2cms

// 		ys, as, rs, gs, s0s, s1s := 0., 0., 0., 0., 0., 0.
// 		for i := range b.cids {
// 			s0s += ws[i].Storage()
// 		}
// 		for i := range b.cids {
// 			y := v[met.AtmosphericYield]   // precipitation/atmospheric yield (rainfall + snowmelt)
// 			ep := v[met.AtmosphericDemand] // evaporative demand

// 			s0 := ws[i].Storage()
// 			a, r, g := ws[i].UpdateWT(y, ep, dm+drel[i])
// 			ws[i].AddToStorage(r * (1. - p0[i]))
// 			r *= p0[i]
// 			s1 := ws[i].Storage()
// 			s1s += s1

// 			wbal := y + s0 - (a + r + g + s1)
// 			if math.Abs(wbal) > nearzero {
// 				fmt.Printf("|wbal| = %e\n", wbal)
// 			}

// 			ys += y
// 			as += a
// 			// rs += r
// 			if ds[i] == -1 { // outlet cell
// 				rs += r
// 			} else {
// 				ws[xr[ds[i]]].AddToStorage(r)
// 			}
// 			gs += g
// 		}
// 		yss += ys
// 		ass += as
// 		rss += rs
// 		gss += gs
// 		hb := Qs * math.Exp(-dm/m)
// 		sim[k] = (rs/b.fncid + hb) * h2cms
// 		bf[k] = hb * h2cms
// 		dm += hb - gs/b.fncid

// 		wbal := ys + s0s - (as + rs + gs + s1s)
// 		if math.Abs(wbal) > nearzero {
// 			fmt.Printf("(sum)|wbal| = %e\n", wbal)
// 		}
// 		k++
// 	}
// 	return
// }

// // evalTest 3: with topology, topmodel gwr (only 1 gw reservoir)
// func (b *subdomain) evalTest(p *sample, Qs, m float64, print bool) (of float64) {
// 	nstep, dtb, dte, intvl := b.frc.trimFrc(-1)
// 	h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor
// 	obs, sim, bf, i := make([]float64, nstep), make([]float64, nstep), make([]float64, nstep), 0
// 	yss, ass, rss, gss := 0., 0., 0., 0.
// 	dm := -m * math.Log(b.frc.Q0/Qs) // initialize mean TOPMODEL deficit
// 	defer func() {
// 		mmio.ObsSim("hyd.png", obs, sim, bf, nil)
// 		of = report(obs, sim, yss, ass, rss, gss, b.fncid, nstep, print)
// 	}()

// 	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
// 		// fmt.Println(d)
// 		v := b.frc.c[d]
// 		obs[i] = v[met.UnitDischarge] * h2cms

// 		ys, as, rs, gs, s0s, s1s := 0., 0., 0., 0., 0., 0.
// 		for _, c := range b.cids {
// 			s0s += p.ws[c].Storage()
// 		}
// 		for _, c := range b.cids {
// 			y := v[met.AtmosphericYield]   // precipitation/atmospheric yield (rainfall + snowmelt)
// 			ep := v[met.AtmosphericDemand] // evaporative demand

// 			s0 := p.ws[c].Storage()
// 			a, r, g := p.ws[c].Update(y, ep)
// 			p.ws[c].AddToStorage(r * (1. - p.p0[c]))
// 			r *= p.p0[c]
// 			s1 := p.ws[c].Storage()
// 			s1s += s1

// 			wbal := y + s0 - (a + r + g + s1)
// 			if math.Abs(wbal) > nearzero {
// 				fmt.Printf("|wbal| = %e\n", wbal)
// 			}

// 			ys += y
// 			as += a
// 			// rs += r
// 			if b.ds[c] == -1 { // outlet cell
// 				rs += r
// 			} else {
// 				p.ws[b.ds[c]].AddToStorage(r)
// 			}
// 			gs += g
// 		}
// 		yss += ys
// 		ass += as
// 		rss += rs
// 		gss += gs
// 		hb := Qs * math.Exp(-dm/m)
// 		sim[i] = (rs/b.fncid + hb) * h2cms
// 		bf[i] = hb * h2cms
// 		dm += hb - gs/b.fncid

// 		wbal := ys + s0s - (as + rs + gs + s1s)
// 		if math.Abs(wbal) > nearzero {
// 			fmt.Printf("(sum)|wbal| = %e\n", wbal)
// 		}
// 		i++
// 	}
// 	return
// }

// // evalTest 2: no topology, topmodel gwr
// func (b *subdomain) evalTest(p *sample, Qs, m float64, print bool) (of float64) {
// 	nstep, dtb, dte, intvl := b.frc.trimFrc(-1)
// 	h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor
// 	obs, sim, bf, i := make([]float64, nstep), make([]float64, nstep), make([]float64, nstep), 0
// 	yss, ass, rss, gss := 0., 0., 0., 0.
// 	defer func() {
// 		mmio.ObsSim("hyd.png", obs, sim, bf, nil)
// 		of = report(obs, sim, yss, ass, rss, gss, b.fncid, nstep, print)
// 	}()

// 	dm := -m * math.Log(b.frc.Q0/Qs) // mean TOPMODEL deficit
// 	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
// 		// fmt.Println(d)
// 		v := b.frc.c[d]
// 		obs[i] = v[met.UnitDischarge] * h2cms

// 		ys, as, rs, gs, s0s, s1s := 0., 0., 0., 0., 0., 0.
// 		for _, c := range b.cids {
// 			s0s += p.ws[c].Storage()
// 		}
// 		for _, c := range b.cids {
// 			y := v[met.AtmosphericYield]   // precipitation/atmospheric yield (rainfall + snowmelt)
// 			ep := v[met.AtmosphericDemand] // evaporative demand

// 			s0 := p.ws[c].Storage()
// 			a, r, g := p.ws[c].Update(y, ep)
// 			p.ws[c].AddToStorage(r * (1. - p.p0[c]))
// 			r *= p.p0[c]
// 			s1 := p.ws[c].Storage()
// 			s1s += s1

// 			wbal := y + s0 - (a + r + g + s1)
// 			if math.Abs(wbal) > nearzero {
// 				fmt.Printf("|wbal| = %e\n", wbal)
// 			}

// 			ys += y
// 			as += a
// 			rs += r
// 			gs += g
// 		}
// 		yss += ys
// 		ass += as
// 		rss += rs
// 		gss += gs
// 		hb := Qs * math.Exp(-dm/m)
// 		sim[i] = (rs/b.fncid + hb) * h2cms
// 		bf[i] = hb * h2cms
// 		dm += hb - gs/b.fncid

// 		wbal := ys + s0s - (as + rs + gs + s1s)
// 		if math.Abs(wbal) > nearzero {
// 			fmt.Printf("(sum)|wbal| = %e\n", wbal)
// 		}
// 		i++
// 	}
// 	return
// }

// // evalTest 1: no topology, linear gwr
// func (b *subdomain) evalTest(p *sample, k, dummy float64, print bool) (of float64) {
// 	nstep, dtb, dte, intvl := b.frc.trimFrc(-1)
// 	h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor
// 	obs, sim, bf, i := make([]float64, nstep), make([]float64, nstep), make([]float64, nstep), 0
// 	yss, ass, rss, gss := 0., 0., 0., 0.
// 	defer func() {
// 		mmio.ObsSim("hyd.png", obs, sim, bf, nil)
// 		of = report(obs, sim, yss, ass, rss, gss, b.fncid, nstep, print)
// 	}()

// 	gw := b.frc.Q0 / k
// 	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
// 		// fmt.Println(d)
// 		v := b.frc.c[d]
// 		obs[i] = v[met.UnitDischarge] * h2cms

// 		ys, as, rs, gs := 0., 0., 0., 0.
// 		for _, c := range b.cids {
// 			y := v[met.AtmosphericYield]   // precipitation/atmospheric yield (rainfall + snowmelt)
// 			ep := v[met.AtmosphericDemand] // evaporative demand

// 			s0 := p.ws[c].Storage()
// 			a, r, g := p.ws[c].Update(y, ep)
// 			p.ws[c].AddToStorage(r * (1. - p.p0[c]))
// 			r *= p.p0[c]
// 			ds := p.ws[c].Storage() - s0

// 			wbal := y - (a + r + g + ds)
// 			if math.Abs(wbal) > nearzero {
// 				fmt.Printf("|wbal| = %e\n", wbal)
// 			}

// 			ys += y
// 			as += a
// 			rs += r
// 			gs += g
// 		}
// 		yss += ys
// 		ass += as
// 		rss += rs
// 		gss += gs
// 		sim[i] = (rs/b.fncid + k*gw) * h2cms
// 		bf[i] = k * gw * h2cms
// 		gw = (1.-k)*gw + gs/b.fncid
// 		i++
// 	}
// 	return
// }
