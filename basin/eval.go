package basin

import (
	"fmt"
	"math"
)

const (
	hx = 0.01
	fe = .75
)

func (p *subsample) eval(Ds, m float64, res resulter, monid []int) {
	ncid := int(p.fncid)
	obs := make(map[int]monitor, len(monid))
	sim, bf, hsto, gsto := make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep)
	// yss, ass, rss, gss, bss := 0., 0., 0., 0., 0.
	// distributed monitors [mm/yr]
	gy, ga, gr, gg, gb := make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid)

	defer func() {
		res.getTotals(sim, bf, hsto, gsto)
		for _, v := range obs {
			go v.print()
		}
		g := gmonitor{gy, ga, gr, gg, gb}
		go g.print(p.ws, p.in, p.xr, p.ds, float64(p.nstep))
	}()

	for _, c := range monid {
		obs[p.xr[c]] = monitor{c: c, v: make([]float64, p.nstep)}
	}

	dm, s0s := p.dm, p.s0s
	for k := 0; k < p.nstep; k++ {
		ys, ins, as, rs, gs, s1s, bs, dm0 := 0., 0., 0., 0., 0., 0., 0., dm
		for i, v := range p.in {
			p.ws[i].AddToStorage(v[k]) // inflow from up sws
			ins += v[k]
		}
		for i := 0; i < ncid; i++ {
			s0 := p.ws[i].Storage()
			y := p.y[k][0]
			ep := p.ep[k][0] * fe
			drel := p.drel[i]
			p0 := p.p0[i]
			a, r, g := p.ws[i].UpdateWT(y, ep, dm+drel)
			x := r * (1. - p0)
			if x > hx {
				x = hx
			}
			p.ws[i].AddToStorage(x)
			r -= x
			// p.ws[i].AddToStorage(r * (1. - p0))
			// r *= p0
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
				p.ws[p.xr[p.ds[i]]].AddToStorage(r)
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
		bf[k] = bs
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
			fmt.Print(".")
		}
		s0s = s1s
	}
	return
}
