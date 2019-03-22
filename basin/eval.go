package basin

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/maseology/goHydro/met"
	"github.com/maseology/mmio"
	"github.com/maseology/objfunc"
)

// evalNoCasc evaluates (runs) the basin model without cascades
func (b *Basin) evalNoCasc(p *sample) (of float64) {
	nstep := b.frc.h.Nstep()
	o, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	defer func() {
		of = 1. - objfunc.KGEi(o, s)
	}()

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
		o[i] = v[met.UnitDischarge] * b.contarea / 86400.0 // cms
		s[i] = rsum * b.contarea / 86400.0
		i++
	}
	return
}

// evalWB same as evalNoCasc() except with rigorous mass balance checking
func (b *Basin) evalWB(p *sample, print bool) (of float64) {
	nstep := b.frc.h.Nstep()
	o, g, x, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	defer func() {
		of = 1. - objfunc.KGEi(o, s)
		if print {
			// C:/Users/mason/OneDrive/R/dygraph/obssim_csv_viewer.R
			mmio.WriteCSV("hydrograph.csv", "date,obs,sim,gw,excess", dt, o, s, g, x)
			// mmio.ObsSim("hydrograph.png", o[730:], s[730:])
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

		if math.Abs(wbal/b.fncid) > 1e-10 {
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
			log.Fatalf(" hru water-balance error, |wbal| = %.3e m", math.Abs(wbal))
		}
		wbalBasin := v[met.AtmosphericYield] - gwlast + slsum - (-p.gw.Dm + ssum + asum + rsum)
		if math.Abs(wbalBasin) > 1e-10 && math.Log10(p.gw.Dm) < 5. {
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
