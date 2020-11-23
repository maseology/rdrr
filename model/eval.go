package model

import (
	"fmt"
	"math"
)

const (
	hx = 0.01
	fe = 1.
)

func eval(p *subsample, Ds, m float64, res resulter, monid []int) {
	ncid := int(p.fncid)
	obs := make(map[int]monitor, len(monid))
	sim, hsto, gsto := make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep)

	defer func() { res.getTotals(sim, hsto, gsto) }()

	for _, c := range monid {
		obs[p.cxr[c]] = monitor{c: c, v: make([]float64, p.nstep)}
	}

	dm := p.dm
	for k := 0; k < p.nstep; k++ {
		rs, gs, s1s, bs := 0., 0., 0., 0.
		for i, v := range p.in {
			p.ws[i].Srf.Sto += v[k] // inflow from up sws
		}
		for i := 0; i < ncid; i++ {
			_, r, g := p.ws[i].UpdateWT(p.y[p.mxr[i]][k], p.ep[p.mxr[i]][k], dm+p.drel[i])
			x := r * (1. - p.p0[i])
			if x > hx {
				x = hx
			}
			p.ws[i].Srf.Sto += x
			r -= x
			s1s += p.ws[i].Storage()

			hb := 0.
			if v, ok := p.strm[i]; ok {
				hb = v * math.Exp((Ds-dm-p.drel[i])/m)
				bs += hb
				r += hb
			}
			if _, ok := obs[i]; ok {
				obs[i].v[k] = r
			}
			if p.ds[i] == -1 { // outlet cell
				rs += r
			} else {
				p.ws[p.cxr[p.ds[i]]].Srf.Sto += r
			}
			gs += g
		}
		dm += (bs - gs) / p.fncid
		sim[k] = rs
		hsto[k] = s1s / p.fncid
		gsto[k] = bs / p.fncid
	}
	return
}

func evalWB(p *subsample, Ds, m float64, res resulter, monid []int) {
	ncid := int(p.fncid)
	obs := make(map[int]monitor, len(monid))
	sim, hsto, gsto := make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep)
	// yss, ass, rss, gss, bss := 0., 0., 0., 0., 0.
	// distributed monitors [mm/yr]
	gy, ga, gr, gg, gb := make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid)

	defer func() {
		res.getTotals(sim, hsto, gsto)
		for _, v := range obs {
			gwg.Add(1)
			go v.print()
		}
		g := gmonitor{gy, ga, gr, gg, gb}
		gwg.Add(1)
		go g.print(p.ws, p.in, p.cxr, p.ds, p.intvl, p.fncid) // float64(p.nstep))
	}()

	for _, c := range monid {
		obs[p.cxr[c]] = monitor{c: c, v: make([]float64, p.nstep)}
	}

	dm, s0s := p.dm, p.s0s
	for k := 0; k < p.nstep; k++ {
		// doy := p.t[k].doy // day of year
		ys, ins, as, rs, gs, s1s, bs, dm0 := 0., 0., 0., 0., 0., 0., 0., dm
		for i, v := range p.in {
			p.ws[i].Srf.Sto += v[k] // inflow from up sws
			ins += v[k]
		}
		for i := 0; i < ncid; i++ {
			s0 := p.ws[i].Storage()
			y := p.y[p.mxr[i]][k]
			ep := p.ep[p.mxr[i]][k] // p.f[i][doy] // p.ep[k][0] // p.f[i][doy] // p.ep[k][0] * p.f[i][doy]
			drel := p.drel[i]
			p0 := p.p0[i]
			a, r, g := p.ws[i].UpdateWT(y, ep, dm+drel)
			x := r * (1. - p0)
			if x > hx {
				x = hx
			}
			p.ws[i].Srf.Sto += x
			r -= x
			s1 := p.ws[i].Storage()
			s1s += s1

			hruwbal := y + s0 - (a + r + g + s1)
			if math.Abs(hruwbal) > nearzero {
				// fmt.Printf("|hruwbal| = %e\n", hruwbal)
				fmt.Print("^")
			}

			ys += y
			as += a
			gy[i] += y
			ga[i] += a
			hb := 0.
			if v, ok := p.strm[i]; ok {
				hb = v * math.Exp((Ds-dm-drel)/m)
				bs += hb
				r += hb
				gb[i] += hb
			}
			if _, ok := obs[i]; ok {
				obs[i].v[k] = r
			}
			if p.ds[i] == -1 { // outlet cell
				rs += r
			} else {
				p.ws[p.cxr[p.ds[i]]].Srf.Sto += r
			}
			gs += g
			gr[i] += r
			gg[i] += g
		}
		// yss += ys
		// ass += as
		// rss += rs
		// gss += gs
		// bss += bs
		dm += (bs - gs) / p.fncid
		sim[k] = rs
		// bf[k] = bs
		hsto[k] = s1s / p.fncid
		gsto[k] = bs / p.fncid // dm

		hruwbal := ys + ins + bs + s0s - (as + rs + gs + s1s)
		if math.Abs(hruwbal) > nearzero {
			// fmt.Printf("(sum)|hruwbal| = %e\n", hruwbal)
			fmt.Print("*")
		}

		basinwbal := ys + ins + (dm-dm0)*p.fncid + s0s - (as + rs + s1s)
		// basinwbal := (dm - dm0) + (gs-bs)/p.fncid // gwbal
		if math.Abs(basinwbal) > nearzero {
			// fmt.Printf("|basinwbal| = %e\n", basinwbal)
			fmt.Print("+")
		}
		s0s = s1s
	}
	return
}

func evalMC(p *subsample, Ds, m float64, res resulter, monid []int) {
	ncid := int(p.fncid)
	obs := make(map[int]monitor, len(monid))
	sim, hsto, gsto := make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep)
	gy, ga, gr, gg, gb := make([][]float64, 12), make([][]float64, 12), make([][]float64, 12), make([][]float64, 12), make([][]float64, 12)
	for i := 0; i < 12; i++ {
		gy[i], ga[i], gr[i], gg[i], gb[i] = make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid)
	}

	defer func() {
		res.getTotals(sim, hsto, gsto)
		for _, v := range obs {
			gwg.Add(1)
			go v.print()
		}
		g := mcmonitor{gy, ga, gr, gg, gb}
		gwg.Add(1)
		go g.print(p.in, p.cxr, p.ds, float64(p.nstep))
	}()

	// defer func() { res.getTotals(sim, hsto, gsto) }()

	for _, c := range monid {
		obs[p.cxr[c]] = monitor{c: c, v: make([]float64, p.nstep)}
	}

	dm := p.dm
	for k := 0; k < p.nstep; k++ {
		mt := p.mt[k] - 1
		rs, gs, s1s, bs := 0., 0., 0., 0.
		for i, v := range p.in {
			p.ws[i].Srf.Sto += v[k] // inflow from up sws
		}
		for i := 0; i < ncid; i++ {
			y := p.y[p.mxr[i]][k]
			a, r, g := p.ws[i].UpdateWT(y, p.ep[p.mxr[i]][k], dm+p.drel[i])
			x := r * (1. - p.p0[i])
			if x > hx {
				x = hx
			}
			p.ws[i].Srf.Sto += x
			r -= x
			s1s += p.ws[i].Storage()

			gy[mt][i] += y
			ga[mt][i] += a
			hb := 0.
			if v, ok := p.strm[i]; ok {
				hb = v * math.Exp((Ds-dm-p.drel[i])/m)
				bs += hb
				r += hb
				gb[mt][i] += hb
			}
			if _, ok := obs[i]; ok {
				obs[i].v[k] = r
			}
			if p.ds[i] == -1 { // outlet cell
				rs += r
			} else {
				p.ws[p.cxr[p.ds[i]]].Srf.Sto += r
			}
			gs += g
			gr[mt][i] += r
			gg[mt][i] += g
		}
		dm += (bs - gs) / p.fncid
		sim[k] = rs
		hsto[k] = s1s / p.fncid
		gsto[k] = bs / p.fncid
	}
	return
}
