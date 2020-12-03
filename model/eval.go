package model

import (
	"math"
)

func eval(p *evaluation, Dinc, m float64, res resulter, monid []int) {
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
		for i, v := range p.sources {
			p.ws[i].Srf.Sto += v[k] // inflow from up sws
		}
		for i := 0; i < ncid; i++ {
			_, r, g := p.ws[i].UpdateWT(p.y[p.mxr[i]][k], p.ep[p.mxr[i]][k], dm+p.drel[i] < 0.)
			p.ws[i].Srf.Sto += r * (1. - p.cascf[i])
			r *= p.cascf[i]
			g += p.ws[i].InfiltrateSurplus()
			s1s += p.ws[i].Storage()

			hb := 0.
			if v, ok := p.strmQs[i]; ok {
				hb = v * math.Exp((Dinc-dm-p.drel[i])/m)
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
