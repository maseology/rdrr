package basin

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/maseology/goHydro/met"
	"github.com/maseology/objfunc"
)

const nearzero = 1e-10

// eval evaluates (runs) the basin model with cascade
func (b *Basin) evalCasc(p *sample) float64 {
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
			di := p.gw.GetDi(c)
			if di < -p.rill { // saturation excess runoff (Di: groundwater deficit)
				di += p.rill
			} else {
				di = 0.
			}
			_, r, g := p.bsn[c].Update(v[met.AtmosphericYield]-di+lag[c], v[met.AtmosphericDemand]*b.mdl.f[c][d.YearDay()-1])

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
		rsum += p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * b.contarea / ts // cms
		s[i] = rsum * b.contarea / ts
		i++
	}
	return 1. - objfunc.KGEi(o, s)
}

// evalNoCasc evaluates (runs) the basin model without cascades
func (b *Basin) evalNoCasc(p *sample) float64 {
	nstep, ts := b.frc.h.Nstep(), 86400.
	o, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0

	// run model
	dtb, dte, intvl := b.frc.h.BeginEndInterval()
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		v := b.frc.c[d]
		rsum, gsum := 0., 0.
		for _, c := range b.cids {
			di := p.gw.GetDi(c)
			if di < -p.rill { // saturation excess runoff (Di: groundwater deficit)
				di += p.rill
			} else {
				di = 0.
			}
			_, r, g := p.bsn[c].Update(v[met.AtmosphericYield]-di, v[met.AtmosphericDemand]*b.mdl.f[c][d.YearDay()-1])
			rsum += r
			gsum += g
		}
		rsum /= b.fncid
		gsum /= b.fncid
		rsum += p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * b.contarea / ts // cms
		s[i] = rsum * b.contarea / ts
		i++
	}
	return 1. - objfunc.KGEi(o, s)
}

// evalCascKineWB same as evalCascKine() except with rigorous mass balance checking
func (b *Basin) evalCascKineWB(p *sample, print bool) (of float64) {
	nstep := b.frc.h.Nstep()
	dtb, dte, intvl := b.frc.h.BeginEndInterval()
	gc := b.mdl.w / float64(intvl)    // grid celerity (w/ts)
	cf := b.contarea / float64(intvl) // q to cms conversion factor
	o, g, x, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	defer func() {
		// of = 1. - objfunc.KGEi(o, s)
		of = objfunc.Krausei(computeMonthly(dt, o, s, float64(intvl), b.contarea))
		if print {
			sumHydrograph(dt, o, s, g, x)
			sumMonthly(dt, o, s, float64(intvl), b.contarea)
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			fmt.Printf("   OF: %.3f  Bias: %.3f\n", 1.-of, objfunc.Biasi(o, s))
		}
	}()

	// initialize
	qi, qo := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid) // inflow this timestep, outflow last timestep
	for _, c := range b.cids {
		qi[c] = 0.
		qo[c] = 0.
	}

	// run model
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		// fmt.Println(d)
		v := b.frc.c[d]
		gwlast := p.gw.Dm
		wbsum, asum, rsum, xsum, gsum, fsum, ssum, slsum, ksum := 0., 0., 0., 0., 0., 0., 0., 0., 0.
		for _, c := range b.cids {
			slast := p.bsn[c].Storage() // total HRU storage at beginning on timestep
			slsum += slast
			di := p.gw.GetDi(c)
			if di < -p.rill { // saturation excess runoff (Di: groundwater deficit)
				di += p.rill
				xsum -= di // saturation excess runoff
				gsum += di // negative recharge
			} else {
				di = 0.
			}
			a, r, g := p.bsn[c].Update(v[met.AtmosphericYield]-di, v[met.AtmosphericDemand]*b.mdl.f[c][d.YearDay()-1])
			if a < 0. {
				log.Fatalf(" hru water-balance error, HRU ET = %.3e mm", a*1000.)
			}
			if r < 0. || math.IsNaN(r) {
				log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm", r*1000.)
			}
			if g < 0. {
				log.Fatalf(" hru water-balance error, HRU potential recharge = %.3e mm", g*1000.)
			}
			asum += a
			gsum += g

			// cascade
			f, d := 0., 0.
			if r > 0 {
				qo[c] = p.p0[c]*qi[c] + p.p1[c]*(qo[c]+gc*r)
			} else {
				d = p.bsn[c].Infiltrability()
				f = d // potential infiltration
				if f < 0. {
					log.Fatalf(" hru water-balance error, HRU potential infiltration = %.3e mm", f*1000.)
				}
				fx := (p.p0[c]*qi[c] + p.p1[c]*qo[c]) / p.p1[c] / gc // max available to infiltrate
				if fx < nearzero {
					fx = 0.
				}
				if f > fx {
					f = fx
				}
				qo[c] = p.p0[c]*qi[c] + p.p1[c]*(qo[c]-gc*f)
				r2 := p.bsn[c].UpdateStorage(f) // add infiltration
				if math.Abs(r2) > nearzero {
					log.Fatalf(" hru water-balance error, HRU infiltration from runon exceeds capacity: f = %.3e mm, x = %.3e mm", f*1000., r2*1000.)
				}
				if qo[c] < -nearzero {
					log.Fatalf(" hru water-balance error, negative runoff computed = %.3e mm", qo[c]/gc*1000.)
				}
			}
			if b.ds[c] == -1 {
				rsum += qo[c] / gc // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case sc[c]=0.
			} else {
				qi[b.ds[c]] += qo[c]
			}

			// waterbalance
			s := p.bsn[c].Storage()
			ki, ko := qi[c]/gc, qo[c]/gc
			// k := (qi[c] - qo[c]) / gc //+ r // "mobile" storage
			ds := s - slast
			wbal := v[met.AtmosphericYield] - di + f - (ds + r + g + a)
			if math.Abs(wbal) > nearzero {
				fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f  slast: %.5f  aet: %.5f  rch: % .5f   ri: %.5f   ro: %.5f\n", v[met.AtmosphericYield], -di, s, slast, a, g, qi[c]/gc, qo[c]/gc)
				log.Fatalf(" cell %d: water-balance error, |wbal| = %.5e m", c, math.Abs(wbal))
			}
			wbsum += wbal
			ssum += s
			fsum += f
			ksum += ki - ko + r
			qi[c] = 0.
		}
		ssum /= b.fncid
		slsum /= b.fncid
		asum /= b.fncid
		rsum /= b.fncid
		xsum /= b.fncid
		gsum /= b.fncid
		fsum /= b.fncid
		ksum /= b.fncid

		bf := p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])
		rsum += bf

		wbsum /= b.fncid
		if math.Abs(wbsum) > nearzero {
			fmt.Printf(" step: %d  rillsto: %.5f  m: %.5f\n", i, p.rill, p.m)
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
			log.Fatalf(" (integrated) hru water-balance error, |wbsum| = %.5e m", math.Abs(wbsum))
		}
		wbalBasin := v[met.AtmosphericYield] - gwlast + slsum - (-p.gw.Dm + ssum + asum + rsum + ksum - fsum)
		if math.Abs(wbalBasin) > nearzero && math.Log10(p.gw.Dm) < 5. {
			fmt.Printf(" step: %d  rillsto: %.5f  m: %.5f  n: %.5f\n", i, p.rill, p.m, p.n)
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gwsto: %.5f  wbal: % .5e\n", slsum, ssum, gwlast, p.gw.Dm, wbalBasin)
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

// evalCascWB same as evalCasc() except with rigorous mass balance checking
func (b *Basin) evalCascWB(p *sample, print bool) (of float64) {
	nstep, ts := b.frc.h.Nstep(), 86400.
	o, g, x, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	defer func() {
		of = 1. - objfunc.KGEi(o, s)
		if print {
			sumHydrograph(dt, o, s, g, x)
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
		gwlast, slaglast := p.gw.Dm, 0.
		for _, v := range lag {
			slaglast += v
		}
		wbsum, asum, rsum, csum, xsum, gsum, ssum, slsum := 0., 0., 0., 0., 0., 0., 0., 0.
		for _, c := range b.cids {
			slast := p.bsn[c].Storage() // initial HRU storage
			slsum += slast
			laglast := lag[c] // runon + stored (mobile) water
			csum += laglast
			di := p.gw.GetDi(c)
			if di < -p.rill { // saturation excess runoff (Di: groundwater deficit)
				di += p.rill
				xsum -= di // saturation excess runoff
				gsum += di // negative recharge
			} else {
				di = 0.
			}
			a, r, g := p.bsn[c].Update(v[met.AtmosphericYield]-di+lag[c], v[met.AtmosphericDemand]*b.mdl.f[c][d.YearDay()-1])
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
			s := p.bsn[c].Storage()
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
		bf := p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])
		rsum += bf

		slag := 0.
		for _, v := range lag {
			slag += v
		}
		slag /= b.fncid
		slaglast /= b.fncid

		wbsum /= b.fncid
		if math.Abs(wbsum) > nearzero {
			fmt.Printf(" step: %d  rillsto: %.5f  m: %.5f\n", i, p.rill, p.m)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
			log.Fatalf(" (integrated) hru water-balance error, |wbsum| = %.5e m", math.Abs(wbsum))
		}
		wbalBasin := v[met.AtmosphericYield] - gwlast + slsum + slaglast - (-p.gw.Dm + ssum + asum + rsum + slag)
		if math.Abs(wbalBasin) > nearzero && math.Log10(p.gw.Dm) < 5. {
			fmt.Printf(" step: %d  rillsto: %.5f  m: %.5f\n", i, p.rill, p.m)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gwsto: %.5f  wbal: % .2e\n", slsum, ssum, gwlast, p.gw.Dm, wbalBasin)
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

// evalNoCascWB same as evalNoCasc() except with rigorous mass balance checking
func (b *Basin) evalNoCascWB(p *sample, print bool) (of float64) {
	nstep := b.frc.h.Nstep()
	o, g, x, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	defer func() {
		of = 1. - objfunc.KGEi(o, s)
		if print {
			sumHydrograph(dt, o, s, g, x)
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			fmt.Printf("  KGE: %.3f  Bias: %.3f\n", 1.-of, objfunc.Biasi(o, s))
		}
	}()

	// run model
	dtb, dte, intvl := b.frc.h.BeginEndInterval()
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		// fmt.Println(d)
		v := b.frc.c[d]
		gwlast := p.gw.Dm
		wbal, asum, rsum, xsum, gsum, ssum, slsum := 0., 0., 0., 0., 0., 0., 0.
		for _, c := range b.cids {
			slast := p.bsn[c].Storage() // initial HRU storage
			slsum += slast
			di := p.gw.GetDi(c)
			if di < -p.rill { // saturation excess runoff (Di: groundwater deficit)
				di += p.rill
				xsum -= di // saturation excess runoff
				gsum += di // negative recharge
			} else {
				di = 0.
			}
			a, r, g := p.bsn[c].Update(v[met.AtmosphericYield]-di, v[met.AtmosphericDemand]*b.mdl.f[c][d.YearDay()-1])
			if a < 0. {
				log.Fatalf(" hru water-balance error, HRU ET = %.3e mm", a*1000.)
			}
			if r < 0. {
				log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm", r*1000.)
			}
			if g < 0. {
				log.Fatalf(" hru water-balance error, HRU potential recharge = %.3e mm", g*1000.)
			}
			s := p.bsn[c].Storage()
			wbal += v[met.AtmosphericYield] - di + slast - (s + g + a + r)
			ssum += s
			asum += a
			rsum += r
			gsum += g
		}
		ssum /= b.fncid
		slsum /= b.fncid
		asum /= b.fncid
		rsum /= b.fncid
		xsum /= b.fncid
		gsum /= b.fncid
		bf := p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])
		rsum += bf

		if math.Abs(wbal/b.fncid) > nearzero {
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
			log.Fatalf(" hru water-balance error, |wbal| = %.3e m", math.Abs(wbal))
		}
		wbalBasin := v[met.AtmosphericYield] - gwlast + slsum - (-p.gw.Dm + ssum + asum + rsum)
		if math.Abs(wbalBasin) > nearzero && math.Log10(p.gw.Dm) < 5. {
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gw: %.5f  wbal: % .2e\n", slsum, ssum, gwlast, p.gw.Dm, wbalBasin)
			fmt.Printf(" step: %d  rillsto: %.5f  m: %.5f\n", i, p.rill, p.m)
			log.Fatalf(" basin water-balance error, |wbalBasin| = %.3e m", math.Abs(wbalBasin))
		}

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * b.contarea / 86400.0 // cms
		g[i] = bf * b.contarea / 86400.0
		x[i] = xsum * b.contarea / 86400.0
		s[i] = rsum * b.contarea / 86400.0
		i++
	}
	return
}
