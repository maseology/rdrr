package basin

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/maseology/goHydro/met"
	mmio "github.com/maseology/mmio"
	"github.com/maseology/objfunc"
)

const nearzero = 1e-10

// evalCascWB same as evalCasc() except with rigorous mass balance checking
func (b *subdomain) evalCascWB(p *sample, freeboard float64, print bool) (of float64) {
	// constants and coefficients
	nstep, dtb, dte, intvl := b.frc.trimFrc(15)
	h2cms := b.contarea / float64(intvl)  // [m/ts] to [m³/s] conversion factor
	af := 365.24 * 1000. / float64(nstep) // aggrate conversion factor [mm/yr]

	// monitors
	// outlet discharge [m³/s]: observes, simulated, baseflow
	o, s, bf, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	// water budget [mm]
	ws, wd, wa, wg, wx, wk := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep)
	// distributed monitors [mm/yr]
	gp, ga, gr, gg, gl := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid)

	defer func() {
		fo, fs := mmio.InterfaceToFloat(o), mmio.InterfaceToFloat(s)
		rmse := objfunc.RMSE(fo, fs)
		of = rmse //(1. - kge) //* (1. - mwr2)
		if print {
			kge := objfunc.KGE(fo, fs)
			mwr2 := objfunc.Krause(computeMonthly(dt, fo, fs, float64(intvl), b.contarea))
			nse := objfunc.NSE(fo, fs)
			bias := objfunc.Bias(fo, fs)
			sumHydrograph(dt, o, s, bf)
			sumHydrographWB(dt, ws, wd, wa, wg, wx, wk)
			sumMonthly(dt, o, s, float64(intvl), b.contarea)
			saveBinaryMap1(gp, "precipitation.rmap")
			saveBinaryMap1(ga, "aet.rmap")
			saveBinaryMap1(gr, "runoff.rmap")
			saveBinaryMap1(gg, "recharge.rmap")
			saveBinaryMap1(gl, "mobile.rmap")
			mmio.ObsSim("hyd.png", fo, fs)
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			fmt.Printf("  KGE: %.3f  NSE: %.3f  mon-wr2: %.3f  Bias: %.3f\n", kge, nse, mwr2, bias)
		}
	}()

	lag := make(map[int]float64, b.ncid) // cell storage and runon capture to be applied at the start of a following timestep
	// initialize cell-based state variables; initialize monitors
	for _, c := range b.cids {
		lag[c] = 0.
		gp[c] = 0.
		ga[c] = 0.
		gr[c] = 0.
		gg[c] = 0.
		// check for consistent gw res mapping
		sid, ok := b.rtr.sws[c]
		if !ok && len(b.rtr.sws) > 0 {
			log.Fatalf(" evalCascWB sws error\n")
		}
		if _, ok := p.gw[sid]; !ok {
			log.Fatalf(" evalCascWB gw[sws] error\n")
		}
	}

	// run model
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		// fmt.Println(d)
		v := b.frc.c[d]

		gwdlast, slaglast := 0., 0.
		ggwsum, ggwcnt := make(map[int]float64, len(p.gw)), make(map[int]float64, len(p.gw))
		for k, v := range p.gw {
			gwdlast += v.Dm * v.Ca
			ggwsum[k] = 0. // sum of recharge for gw res k
			ggwcnt[k] = 0. // count of recharge for gw res k
		}
		gwdlast /= b.contarea // basin groundwater deficit at beginning of timestep
		for _, v := range lag {
			slaglast += v
		}

		wbsum, ysum, asum, rsum, csum, xsum, gsum, ssum, slsum, bfsum := 0., 0., 0., 0., 0., 0., 0., 0., 0., 0.
		for _, c := range b.cids {
			y := v[met.AtmosphericYield]     // precipitation/atmospheric yield (rainfall + snowmelt)
			ep := v[met.AtmosphericDemand]   // evaporative demand
			ep *= b.strc.f[c][d.YearDay()-1] // adjust for slope-aspect

			slast := p.ws[c].Storage() // initial HRU storage
			slsum += slast
			laglast := lag[c] // runon + stored (mobile) water
			csum += laglast

			// groundwater discharge
			sid := b.rtr.sws[c]
			di := p.gw[sid].GetDi(c)
			if di < -freeboard { // saturation excess runoff (Di: groundwater deficit)
				di += freeboard
				xsum -= di        // saturation excess runoff
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
			a := p.ws[c].UpdateEp(ep) // aet
			if a < 0. {
				log.Fatalf(" hru water-balance error, HRU ET = %.3e mm\n", a*1000.)
			}
			if r < 0. {
				log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm\n", r*1000.)
			}
			if g < 0. {
				log.Fatalf(" hru water-balance error, HRU potential recharge = %.3e mm\n", g*1000.)
			}
			asum += a
			ggwsum[sid] += g // sum recharge
			ggwcnt[sid]++    // count recharge

			// pre-runoff waterbalance
			s := p.ws[c].Storage()
			wbal := y - di + slast + laglast - (s + g + a)

			// pre-runoff summations
			ysum += y              // sum precipitation (rainfall + snowmelt)
			gp[c] += y * af        // sum grid precip monitor [mm/yr]
			ga[c] += a * af        // sum grid AET [mm/yr]
			gg[c] += (g + di) * af // sum grid recharge [mm/yr]; -di = groundwater discharge

			// cascade
			if b.ds[c] == -1 { // outlet cell
				if _, ok := p.gw[c]; !ok {
					fmt.Printf(" model error: outlet not assigned a groundwater reservoir")
				}
				hbf := p.gw[c].Update(ggwsum[sid] / ggwcnt[sid]) // baseflow from gw[c] discharging to cell c [m/ts]
				bfsum += hbf * p.swsr[c]                         // basin baseflow [m/ts] (area-corrected)
				rsum += r + hbf*p.celr[c]                        // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case p.pa[c]=0.
				lag[c] = 0.
				wbal -= r
				gr[c] += r * 1000.
			} else {
				if _, ok := p.gw[c]; ok {
					hbf := p.gw[c].Update(ggwsum[sid] / ggwcnt[sid]) // baseflow from gw[c] discharging to cell c [m/ts]
					bfsum += hbf * p.swsr[c]                         // basin baseflow [m/ts] (area-corrected)
					lag[b.ds[c]] += hbf * p.celr[c]                  // adding baseflow to input of downstream cell [m/ts]
				}
				rt := r * p.p0[c]
				lag[c] = r * (1. - p.p0[c]) // retention
				if lag[c] > 1. {
					rt += lag[c] - 1.
					lag[c] = 1.
				}
				lag[b.ds[c]] += rt
				wbal -= rt + lag[c]
				gr[c] += rt * 1000.
			}

			if math.Abs(wbal) > nearzero {
				fmt.Printf(" step: %d  cell ID: %d\n", i, c)
				fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f  slast: %.5f  aet: %.5f  rch: % .5f   ro: %.5f\n", y, -di, s, slast, a, g, r*p.p0[c])
				log.Fatalf(" cell %d: water-balance error, |wbal| = %.5e m\n", c, math.Abs(wbal))
			}
			wbsum += wbal
			ssum += s
		}
		ysum /= b.fncid
		ssum /= b.fncid
		slsum /= b.fncid
		asum /= b.fncid
		rsum /= b.fncid
		csum /= b.fncid
		xsum /= b.fncid
		gsum /= b.fncid

		slag := 0.
		for k, v := range lag {
			slag += v
			gl[k] = v * 1000.
		}
		slag /= b.fncid
		slaglast /= b.fncid

		gwd := 0.
		for _, v := range p.gw {
			gwd += v.Dm * v.Ca
		}
		gwd /= b.contarea // current basin groundwater deficit

		wbsum /= b.fncid
		if math.Abs(wbsum) > nearzero {
			fmt.Printf(" step: %d  freeboard: %.5f\n", i, freeboard)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
			log.Fatalf(" (integrated) hru water-balance error, |wbsum| = %.5e m\n", math.Abs(wbsum))
		}
		wbalBasin := ysum - gwdlast + slsum + slaglast - (-gwd + ssum + asum + rsum + slag)
		if math.Abs(wbalBasin) > nearzero {
			fmt.Printf(" step: %d  freeboard: %.5f\n", i, freeboard)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gwsto: %.5f  wbalBasin: % .2e\n", slsum, ssum, gwdlast, gwd, wbalBasin)
			log.Fatalf(" basin water-balance error, |wbalBasin| = %.5e m\n", math.Abs(wbalBasin))
		}

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * h2cms
		bf[i] = bfsum * h2cms // groundwater discharge to streams [m³/s]
		// x[i] = xsum * h2cms
		s[i] = rsum * h2cms
		ws[i] = ssum * 1000. // CE storage
		wd[i] = gwd * 1000.  // GW deficit
		wg[i] = gsum * 1000. // groundwater recharge
		wx[i] = xsum * 1000. // saturation excess runoff
		wk[i] = slag * 1000. // mobile runoff
		wa[i] = asum * 1000. // evaporation
		i++
	}
	return
}
