package model

import "math"

func evalMC(p *evaluation, res resulter, monid []int) {
	ncid := int(p.fncid)
	obs, fcms := make(map[int]monitor, len(monid)), p.ca/p.intvl
	sim, hsto, gsto := make([]float64, p.nstep), make([]float64, p.nstep), make([]float64, p.nstep)
	gy, ga, gr, gg, gb := make([][]float64, 12), make([][]float64, 12), make([][]float64, 12), make([][]float64, 12), make([][]float64, 12)
	for i := 0; i < 12; i++ {
		gy[i], ga[i], gr[i], gg[i], gb[i] = make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid)
	}

	for _, c := range monid {
		obs[p.cxr[c]] = monitor{c: c, v: make([]float64, p.nstep)}
	}

	dm := p.dm
	for k := 0; k < p.nstep; k++ {
		mt := p.mt[k] - 1
		rs, gs, s1s, bs := 0., 0., 0., 0.
		for i, v := range p.sources {
			p.ws[i].Sdet.Sto += v[k] // inflow from up sws
		}
		for i := 0; i < ncid; i++ {
			y := p.y[p.mxr[i]][k]
			a, r, g := p.ws[i].UpdateWT(y, p.ep[p.mxr[i]][k], dm+p.drel[i] < 0.)

			p.ws[i].Sdet.Sto += r * (1. - p.cascf[i])
			r *= p.cascf[i]
			g += p.ws[i].InfiltrateSurplus()
			s1s += p.ws[i].Storage()

			if v, ok := p.strmQs[i]; ok {
				b := v * math.Exp((-dm-p.drel[i])/p.m[p.gxr[i]])
				bs += b
				r += b
				gb[mt][i] += b
			}

			gy[mt][i] += y
			ga[mt][i] += a

			if _, ok := obs[i]; ok {
				obs[i].v[k] = r * fcms
			}
			if p.ds[i] == -1 { // outlet cell
				rs += r
			} else {
				p.ws[p.cxr[p.ds[i]]].Sdet.Sto += r
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

	func() {
		res.getTotals(sim, hsto, gsto)
		for _, v := range obs {
			gwg.Add(1)
			go v.print(p.dir)
		}
		g := mcmonitor{gy, ga, gr, gg, gb}
		gwg.Add(1)
		go g.print(p.sources, p.cxr, p.ds, float64(p.nstep), p.dir)
	}()
	return
}
