package basin

import (
	"fmt"
	"time"

	"github.com/maseology/goHydro/met"
	mmio "github.com/maseology/mmio"
	"github.com/maseology/objfunc"
)

// evalCasc runs rdrr in cascade mode
func (b *subdomain) evalCasc(p *sample, freeboard float64, print bool) (of float64) {
	// constants and coefficients
	nstep := b.frc.h.Nstep()                      // number of time steps
	dtb, dte, intvl := b.frc.h.BeginEndInterval() // start date, end date, time step interval [s]
	dur := dte.Sub(dtb)
	if dur > 15*365*86400*time.Second {
		dtb = dte.Add(-15 * 365 * 86400 * time.Second)
	}
	h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor

	swsr, celr := make(map[int]float64, len(p.gw)), make(map[int]float64, len(p.gw))
	for k, v := range p.gw {
		swsr[k] = v.Ca / b.contarea // groundwatershed area to catchment area
		celr[k] = v.Ca / b.strc.a   // groundwatershed area to cell area
	}

	// monitors
	// outlet discharge [m³/s]: observes, simulated, baseflow
	o, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0

	defer func() {
		fo, fs := mmio.InterfaceToFloat(o), mmio.InterfaceToFloat(s)
		rmse := objfunc.RMSE(fo, fs)
		of = rmse //(1. - kge) //* (1. - mwr2)
		if print {
			kge := objfunc.KGE(fo, fs)
			mwr2 := objfunc.Krause(computeMonthly(dt, fo, fs, float64(intvl), b.contarea))
			nse := objfunc.NSE(fo, fs)
			bias := objfunc.Bias(fo, fs)
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			fmt.Printf("  KGE: %.3f  NSE: %.3f  mon-wr2: %.3f  Bias: %.3f\n", kge, nse, mwr2, bias)
		}
	}()

	lag := make(map[int]float64, b.ncid) // cell storage and runon capture to be applied at the start of a following timestep
	// initialize cell-based state variables; initialize monitors
	for _, c := range b.cids {
		lag[c] = 0.
	}

	// run model
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		// fmt.Println(d)
		v := b.frc.c[d]

		ggwsum, ggwcnt := make(map[int]float64, len(p.gw)), make(map[int]float64, len(p.gw))
		for k := range p.gw {
			ggwsum[k] = 0. // sum of recharge for gw res k
			ggwcnt[k] = 0. // count of recharge for gw res k
		}

		rsum := 0.
		for _, c := range b.cids {
			y := v[met.AtmosphericYield]     // precipitation/atmospheric yield (rainfall + snowmelt)
			ep := v[met.AtmosphericDemand]   // evaporative demand
			ep *= b.strc.f[c][d.YearDay()-1] // adjust for slope-aspect

			// groundwater discharge
			sid := b.mpr.sws[c]
			di := p.gw[sid].GetDi(c)
			if di < -freeboard { // saturation excess runoff (Di: groundwater deficit)
				di += freeboard
				ggwsum[sid] += di // negative recharge (groundwater discharge) [m]
			} else {
				di = 0.
			}

			// update HRU
			r := p.ws[c].UpdateP(y - di + lag[c]) // runoff
			g := 0.                               // recharge
			if di >= 0. {                         // only recharge when deficit is available; otherwise reject
				g = p.ws[c].UpdatePerc()
			}
			p.ws[c].UpdateEp(ep) // aet
			ggwsum[sid] += g     // sum recharge
			ggwcnt[sid]++        // count recharge

			// cascade
			if b.ds[c] == -1 { // outlet cell
				if _, ok := p.gw[c]; !ok {
					fmt.Printf(" model error: outlet not assigned a groundwater reservoir")
				}
				hbf := p.gw[c].Update(ggwsum[sid] / ggwcnt[sid]) // baseflow from gw[c] discharging to cell c [m/ts]
				rsum += r + hbf*celr[c]                          // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case p.pa[c]=0.
				lag[c] = 0.
			} else {
				if _, ok := p.gw[c]; ok {
					hbf := p.gw[c].Update(ggwsum[sid] / ggwcnt[sid]) // baseflow from gw[c] discharging to cell c [m/ts]
					lag[b.ds[c]] += hbf * celr[c]                    // adding baseflow to input of downstream cell [m/ts]
				}
				rt := r * p.p0[c]
				lag[c] = r * (1. - p.p0[c]) // retention
				if lag[c] > 1. {
					rt += lag[c] - 1.
					lag[c] = 1.
				}
				lag[b.ds[c]] += rt
			}
		}
		rsum /= b.fncid

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * h2cms
		s[i] = rsum * h2cms
		i++
	}
	return
}
