package basin

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/maseology/goHydro/met"
	"github.com/maseology/objfunc"
)

// evalCasc evaluates (runs) the basin model with cascade
func (b *subdomain) evalCasc(p *sample, rill float64) float64 {
	nstep, ts := b.frc.h.Nstep(), 86400.
	o, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	lag := make(map[int]float64, b.ncid) // cell storage and runon capture to be applied at the start of a following timestep
	for _, c := range b.cids {
		lag[c] = 0.
	}

	// run model
	dtb, dte, intvl := b.frc.h.BeginEndInterval()
	cascfrac := 0.1
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		v := b.frc.c[d]
		rsum, gsum := 0., 0.

		for _, c := range b.cids {
			di := p.gw[0].GetDi(c)
			if di < -rill { // saturation excess runoff (Di: groundwater deficit)
				di += rill
			} else {
				di = 0.
			}
			_, r, g := p.ws[c].Update(v[met.AtmosphericYield]-di+lag[c], v[met.AtmosphericDemand]*b.strc.f[c][d.YearDay()-1])

			// cascade
			gsum += g
			if b.ds[c] == -1 {
				rsum += r
			} else {
				lag[b.ds[c]] = r * cascfrac
				lag[c] = r * (1. - cascfrac)
			}
		}
		rsum /= b.fncid
		gsum /= b.fncid
		rsum += p.gw[0].Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * b.contarea / ts // cms
		s[i] = rsum * b.contarea / ts
		i++
	}
	return 1. - objfunc.KGEi(o, s)
}

// evalCascWB same as evalCasc() except with rigorous mass balance checking
func (b *subdomain) evalCascWB(p *sample, rill float64, print bool) (of float64) {
	nstep, ts := b.frc.h.Nstep(), 86400.
	o, g, x, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	defer func() {
		of = 1. - objfunc.KGEi(o, s)
		if print {
			sumHydrograph(dt, o, s, g)
			sumMonthly(dt, o, s, ts, b.contarea)
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			fmt.Printf("  KGE: %.3f  Bias: %.3f\n", 1.-of, objfunc.Biasi(o, s))
		}
	}()
	lag := make(map[int]float64, b.ncid) // cell storage and runon capture to be applied at the start of a following timestep
	for _, c := range b.cids {
		lag[c] = 0.
	}

	// run model
	dtb, dte, intvl := b.frc.h.BeginEndInterval()
	cf := b.contarea / ts // q to cms conversion factor
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		// fmt.Println(d)
		v := b.frc.c[d]
		gwlast, slaglast := p.gw[0].Dm, 0.
		for _, v := range lag {
			slaglast += v
		}
		wbsum, asum, rsum, csum, xsum, gsum, ssum, slsum := 0., 0., 0., 0., 0., 0., 0., 0.
		for _, c := range b.cids {
			slast := p.ws[c].Storage() // initial HRU storage
			slsum += slast
			laglast := lag[c] // runon + stored (mobile) water
			csum += laglast
			di := p.gw[0].GetDi(c)
			if di < -rill { // saturation excess runoff (Di: groundwater deficit)
				di += rill
				xsum -= di // saturation excess runoff
				gsum += di // negative recharge
			} else {
				di = 0.
			}
			a, r, g := p.ws[c].Update(v[met.AtmosphericYield]-di+lag[c], v[met.AtmosphericDemand]*b.strc.f[c][d.YearDay()-1])
			if a < 0. {
				log.Fatalf(" hru water-balance error, HRU ET = %.3e mm", a*1000.)
			}
			if r < 0. {
				log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm", r*1000.)
			}
			if g < 0. {
				log.Fatalf(" hru water-balance error, HRU potential recharge = %.3e mm", g*1000.)
			}
			asum += a
			gsum += g
			s := p.ws[c].Storage()
			wbal := v[met.AtmosphericYield] - di + slast + laglast - (s + g + a)
			// cascade
			if b.ds[c] == -1 {
				rsum += r // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case p.pa[c]=0.
				lag[c] = 0.
				wbal -= r
			} else {
				lag[b.ds[c]] += r * p.p0[c]
				lag[c] = r * (1. - p.p0[c]) // retention
				wbal -= r*p.p0[c] + lag[c]
			}
			if math.Abs(wbal) > nearzero {
				fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f  slast: %.5f  aet: %.5f  rch: % .5f   ro: %.5f\n", v[met.AtmosphericYield], -di, s, slast, a, g, r*p.p0[c])
				log.Fatalf(" cell %d: water-balance error, |wbal| = %.5e m", c, math.Abs(wbal))
			}
			wbsum += wbal
			ssum += s
		}
		ssum /= b.fncid
		slsum /= b.fncid
		asum /= b.fncid
		rsum /= b.fncid
		csum /= b.fncid
		xsum /= b.fncid
		gsum /= b.fncid
		bf := p.gw[0].Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])
		rsum += bf

		slag := 0.
		for _, v := range lag {
			slag += v
		}
		slag /= b.fncid
		slaglast /= b.fncid

		wbsum /= b.fncid
		if math.Abs(wbsum) > nearzero {
			fmt.Printf(" step: %d  rillsto: %.5f  m: %.5f\n", i, rill, p.gw[0].M)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
			log.Fatalf(" (integrated) hru water-balance error, |wbsum| = %.5e m", math.Abs(wbsum))
		}
		wbalBasin := v[met.AtmosphericYield] - gwlast + slsum + slaglast - (-p.gw[0].Dm + ssum + asum + rsum + slag)
		if math.Abs(wbalBasin) > nearzero && math.Log10(p.gw[0].Dm) < 5. {
			fmt.Printf(" step: %d  rillsto: %.5f  m: %.5f\n", i, rill, p.gw[0].M)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gwsto: %.5f  wbal: % .2e\n", slsum, ssum, gwlast, p.gw[0].Dm, wbalBasin)
			log.Fatalf(" basin water-balance error, |wbalBasin| = %.5e m", math.Abs(wbalBasin))
		}

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * cf
		g[i] = bf * cf
		x[i] = xsum * cf
		s[i] = rsum * cf
		i++
	}
	return
}
