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

const nearzero = 1e-5 //1e-10

// evalCascWB same as evalCasc() except with rigorous mass balance checking
func (b *subdomain) evalCascWB(p *sample, freeboard float64, print bool) (of float64) {
	// constants and coefficients
	nstep, dtb, dte, intvl := b.frc.trimFrc(15)
	h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor
	// af := 365.24 * 1000. / float64(nstep) // aggrate conversion factor [mm/yr]

	// monitors
	// outlet discharge [m³/s]: observes, simulated, baseflow
	o, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	// water budget [mm]
	sy, sa, sr, sg := 0., 0., 0., 0.
	wy, ws, wd, wa, wg, wx, wk := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep)
	// // distributed monitors [mm/yr]
	// gp, ga, gr, gg, gl := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid)

	defer func() {
		fo, fs := mmio.InterfaceToFloat(o), mmio.InterfaceToFloat(s)
		rmse := objfunc.RMSE(fo, fs)
		of = rmse //(1. - kge) //* (1. - mwr2)
		if print {
			kge := objfunc.KGE(fo, fs)
			mwr2 := objfunc.Krause(computeMonthly(dt, fo, fs, float64(intvl), b.contarea))
			nse := objfunc.NSE(fo, fs)
			bias := objfunc.Bias(fo, fs)
			// sumHydrograph(dt, o, s, bf)
			// sumHydrographWB(dt, ws, wd, wa, wg, wx, wk)
			// sumMonthly(dt, o, s, float64(intvl), b.contarea)
			// saveBinaryMap1(gp, "precipitation.rmap")
			// saveBinaryMap1(ga, "aet.rmap")
			// saveBinaryMap1(gr, "runoff.rmap")
			// saveBinaryMap1(gg, "recharge.rmap")
			// saveBinaryMap1(gl, "mobile.rmap")
			mmio.ObsSim("hyd.png", fo, fs)
			sumPlotHydrographWB("wb.png", ws, wd, wk, wx, wa, wg)
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			ff := 365.24 * 1000. / float64(i)
			fmt.Printf("  waterbudget [mm/yr]: pre: %.0f  aet: %.0f  rch: %.0f  ro: %.0f  dif: %.1f\n", sy*ff, sa*ff, sg*ff, sr*ff, (sy-(sa+sg+sr))*ff)
			fmt.Printf("  KGE: %.3f  NSE: %.3f  mon-wr2: %.3f  Bias: %.3f\n", kge, nse, mwr2, bias)
		}
	}()

	lag := make(map[int]float64, b.ncid) // cell storage and runon capture to be applied at the start of a following timestep
	// initialize cell-based state variables; initialize monitors
	for _, c := range b.cids {
		lag[c] = 0.
		// gp[c] = 0.
		// ga[c] = 0.
		// gr[c] = 0.
		// gg[c] = 0.
		// check for consistent gw res mapping
		sid, ok := b.rtr.sws[c]
		if !ok && len(b.rtr.sws) > 0 {
			log.Fatalf(" evalCascWB sws error: subwatersheds have not been loaded\n")
		}
		if _, ok := p.gw[sid]; !ok {
			log.Fatalf(" evalCascWB gw[sws] error: no groundwater reservoir associated with subwatershed %d\n", sid)
		}
	}

	// run model
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		// fmt.Println(d)
		v := b.frc.c[d]

		gwdlast, slaglast := 0., 0.
		ggwsum, ggwcnt := make(map[int]float64, len(p.gw)), make(map[int]float64, len(p.gw))
		for k, v := range p.gw {
			gwdlast += v.Dm * p.swsr[k] // basin groundwater deficit at beginning of timestep
			ggwsum[k] = 0.              // sum of recharge for gw res k
			ggwcnt[k] = 0.              // count of recharge for gw res k
		}
		for _, v := range lag {
			slaglast += v
		}

		wbsum, ysum, asum, rsum, xsum, gsum, ssum, slsum := 0., 0., 0., 0., 0., 0., 0., 0.
		for _, c := range b.cids {
			y := v[met.AtmosphericYield]     // precipitation/atmospheric yield (rainfall + snowmelt)
			ep := v[met.AtmosphericDemand]   // evaporative demand
			ep *= b.strc.f[c][d.YearDay()-1] // adjust for slope-aspect

			slast := p.ws[c].Storage() // initial HRU storage
			slsum += slast
			laglast := lag[c] // runon + stored (mobile) water

			// groundwater discharge
			sid := b.rtr.sws[c]
			di, hb := p.gw[sid].GetDi(c), 0.
			if v, ok := p.gw[sid].Qs[c]; ok { // lateral discharge to streams [m/ts]
				hb = v * math.Exp(-di/p.gw[sid].M)
			}
			if di < -freeboard { // saturation excess runoff (Di: groundwater deficit)
				// if v, ok := p.gw[sid].Qs[c]; ok { // lateral discharge to streams [m]
				// 	hb = v // capping at Di=0 as excess is being handled by the SMA
				// }
				di += freeboard
				xsum -= di // saturation excess runoff
			} else {
				// if v, ok := p.gw[sid].Qs[c]; ok { // lateral discharge to streams [m]
				// 	hb = v * math.Exp(-di/p.gw[sid].M)
				// }
				di = 0.
			}

			// update HRU
			a, r, g := p.ws[c].Update(y-di+lag[c]+hb, ep)
			// r := p.ws[c].UpdateP(y - di + lag[c] + hb) // runoff
			// g := 0.                                    // recharge
			// if di >= 0. {                              // only recharge when deficit is available; otherwise reject
			// 	g = p.ws[c].UpdatePerc()
			// }
			// a := p.ws[c].UpdateEp(ep) // aet
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
			gsum += g
			ggwsum[sid] += g + di - hb // sum recharge less discharge [m]
			ggwcnt[sid]++              // count recharge

			// pre-runoff waterbalance
			s := p.ws[c].Storage()
			wbal := y - di + hb + slast + laglast - (s + g + a)

			// pre-runoff summations
			ysum += y // sum precipitation (rainfall + snowmelt)
			// gp[c] += y * af        // sum grid precip monitor [mm/yr]
			// ga[c] += a * af        // sum grid AET [mm/yr]
			// gg[c] += (g + di - qb) * af // sum grid recharge [mm/yr]; -di = groundwater discharge

			// cascade
			if b.ds[c] == -1 { // outlet cell
				if _, ok := p.gw[c]; !ok {
					fmt.Printf(" model error: outlet not assigned a groundwater reservoir")
				}
				// hbf := p.gw[c].Update(ggwsum[sid] / ggwcnt[sid]) // baseflow from gw[c] discharging to cell c [m/ts]
				p.gw[c].Dm -= ggwsum[sid] / ggwcnt[sid]
				// bfsum += hbf * p.swsr[c]                         // basin baseflow [m/ts] (area-corrected)
				rsum += r //+ hbf*p.celr[c] // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case p.pa[c]=0.
				lag[c] = 0.
				wbal -= r
				// gr[c] += r * 1000.
			} else {
				if _, ok := p.gw[c]; ok {
					p.gw[c].Dm -= ggwsum[sid] / ggwcnt[sid]
					// hbf := p.gw[c].Update(ggwsum[sid] / ggwcnt[sid]) // baseflow from gw[c] discharging to cell c [m/ts]
					// bfsum += hbf * p.swsr[c]                         // basin baseflow [m/ts] (area-corrected)
					// lag[b.ds[c]] += hbf * p.celr[c] // adding baseflow to input of downstream cell [m/ts]
				}
				rt := r * p.p0[c]
				lag[c] = r * (1. - p.p0[c]) // retention
				if lag[c] > 1. {
					rt += lag[c] - 1.
					lag[c] = 1.
				}
				lag[b.ds[c]] += rt
				wbal -= rt + lag[c]
				// gr[c] += rt * 1000.
			}

			if math.Abs(wbal) > nearzero {
				fmt.Printf(" step: %d  cell ID: %d  Topm: %f\n", i, c, p.gw[sid].M)
				fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f  slast: %.5f  aet: %.5f  rch: % .5f   ro: %.5f\n", y, -di, s, slast, a, g, r*p.p0[c])
				fmt.Printf(" cell %d: water-balance error, |wbal| = %.5e m\n", c, math.Abs(wbal))
			}
			wbsum += wbal
			ssum += s
		}
		ysum /= b.fncid
		ssum /= b.fncid
		slsum /= b.fncid
		asum /= b.fncid
		rsum /= b.fncid
		xsum /= b.fncid
		gsum /= b.fncid

		slag := 0.
		for _, v := range lag {
			slag += v
			// gl[k] = v * 1000.
		}
		slag /= b.fncid
		slaglast /= b.fncid

		gwd := 0. // current basin groundwater deficit
		for k, v := range p.gw {
			gwd += v.Dm * p.swsr[k]
		}

		wbsum /= b.fncid
		if math.Abs(wbsum) > nearzero {
			fmt.Printf(" step: %d  freeboard: %.5f\n", i, freeboard)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" (integrated) hru water-balance error, |wbsum| = %.5e m\n", math.Abs(wbsum))
		}
		// wbalBasin := ysum + bfsum + xsum + slsum + slaglast - (ssum + asum + rsum + gsum + slag)

		wbalBasinSto := func() float64 {
			gain, loss := ysum, asum+rsum
			s0, s1 := -gwdlast+slsum+slaglast, -gwd+ssum+slag
			return gain + s0 - (loss + s1)
		}()
		if math.Abs(wbalBasinSto) > nearzero {
			fmt.Printf(" step: %d  freeboard: %.5f\n", i, freeboard)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gwsto: %.5f  wbalBasinSto: % .10f\n", slsum, ssum, gwdlast, gwd, wbalBasinSto)
			fmt.Printf(" basin water-balance error, |wbalBasinSto| = %.5e m\n", math.Abs(wbalBasinSto))
		}

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * h2cms
		s[i] = rsum * h2cms
		// bf[i] = bfsum * h2cms // groundwater discharge to streams [m³/s]
		// x[i] = xsum * h2cms
		sy += ysum
		sa += asum
		sr += rsum
		sg += gsum
		wy[i] = ysum * 1000. // yield (rainfall + snowmelt)
		ws[i] = ssum * 1000. // CE storage
		wd[i] = gwd          // GW deficit [m]
		wg[i] = gsum * 1000. // groundwater recharge
		wx[i] = xsum * 1000. // saturation excess runoff
		wk[i] = slag * 1000. // mobile runoff
		wa[i] = asum * 1000. // evaporation
		i++
	}
	return
}
