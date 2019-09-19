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

// evalCascWB same as evalCasc() except with rigorous mass balance checking
func (b *subdomain) evalCascWB(p *sample, Qo, freeboard float64, print bool) (of float64) {
	// constants and coefficients
	const sb = 365 // timesteps for spin-up
	kill := false
	nstep, dtb, dte, intvl := b.frc.trimFrc(-1)
	h2cms := b.contarea / float64(intvl)  // [m/ts] to [m³/s] conversion factor
	af := 365.24 * 1000. / float64(nstep) // aggrate conversion factor [mm/yr]

	// monitors
	// outlet discharge [m³/s]: observes, simulated, baseflow
	o, s, bf, xs, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	// water budget [mm]
	sy, sa, sr, sg := 0., 0., 0., 0.
	wy, ws, wd, wa, wg, wx, wk := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep)
	// distributed monitors [mm/yr]
	gp, ga, gr, gg, gl := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid)

	defer func() {
		if kill {
			of = 9999.
		} else {
			fo, fs := mmio.InterfaceToFloat(o)[sb:], mmio.InterfaceToFloat(s)[sb:]
			rmse := objfunc.RMSE(fo, fs)
			of = rmse //(1. - kge) //* (1. - mwr2)
			if print {
				kge := objfunc.KGE(fo, fs)
				mwr2 := objfunc.Krause(computeMonthly(dt[sb:], fo, fs, float64(intvl), b.contarea))
				nse := objfunc.NSE(fo, fs)
				bias := objfunc.Bias(fo, fs)
				// sumHydrographWB(dt, ws, wd, wa, wg, wx, wk)
				// sumMonthly(dt, o, s, float64(intvl), b.contarea)
				mmio.WriteRMAP("precipitation.rmap", gp, false)
				mmio.WriteRMAP("aet.rmap", ga, false)
				mmio.WriteRMAP("runoff.rmap", gr, false)
				mmio.WriteRMAP("gwe.rmap", gg, false)
				mmio.WriteRMAP("mobile.rmap", gl, false)
				mmio.ObsSim("hyd.png", fo, fs, mmio.InterfaceToFloat(bf)[sb:], mmio.InterfaceToFloat(xs)[sb:])
				sumPlotHydrographWB("wb.png", ws, wd, wk, wx, wa, wg)
				fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
				ff := 365.24 * 1000. / float64(i)
				fmt.Printf("  waterbudget [mm/yr]: pre: %.0f  aet: %.0f  rch: %.0f  ro: %.0f  dif: %.1f\n", sy*ff, sa*ff, sg*ff, sr*ff, (sy-(sa+sg+sr))*ff)
				fmt.Printf("  KGE: %.3f  NSE: %.3f  RMSE: %.3f  mon-wR²: %.3f  Bias: %.3f\n", kge, nse, rmse, mwr2, bias)
			}	
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

		gwdlast, slaglast, ssumlast := 0., 0., 0.
		gsumsws, gcntsws := make(map[int]float64, len(p.gw)), make(map[int]float64, len(p.gw))
		for k, v := range p.gw {
			gwdlast += v.Dm * p.swsr[k] // basin groundwater deficit at beginning of timestep
			gsumsws[k] = 0.             // sum of recharge for gw res k
			gcntsws[k] = 0.             // count of recharge for gw res k
		}
		for _, v := range lag {
			slaglast += v
		}

		wbsum, ysum, asum, bfsum, spsum, rsum, xsum, gsum, ssum := 0., 0., 0., 0., 0., 0., 0., 0., 0.
		for _, c := range b.cids {
			sid := b.rtr.sws[c]              // groundwatershed id
			y := v[met.AtmosphericYield]     // precipitation/atmospheric yield (rainfall + snowmelt)
			ep := v[met.AtmosphericDemand]   // evaporative demand
			ep *= b.strc.f[c][d.YearDay()-1] // adjust for slope-aspect
			di := 0. // p.gw[sid].GetDi(c)

			slast := p.ws[c].Storage() // initial HRU storage
			ssumlast += slast

			a, r, g := p.ws[c].UpdateWT(y+lag[c], ep, di)
			if a < 0. {
				log.Fatalf(" hru water-balance error, HRU ET = %.3e mm\n", a*1000.)
			}
			if r < 0. {
				log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm\n", r*1000.)
			}
			ysum += y         // sum precipitation (rainfall + snowmelt) [m]
			asum += a         // sum actual ET [m]
			if g > 0. {
				gsum += g         // sum recharge [m]
			} else {
				xsum -= g         // sum discharge [m]
			}			
			gsumsws[sid] += g // sum recharge [m], on a sws basis
			gcntsws[sid]++    // count recharge
			gp[c] += y * af   // sum grid precip monitor [mm/yr]
			ga[c] += a * af   // sum grid AET [mm/yr]
			gg[c] += g * af   // sum grid recharge [mm/yr]

			// pre-runoff (hru) waterbalance
			s := p.ws[c].Storage()
			ssum += s
			func() {
				in := y + lag[c]
				out := a + g + r
				delsto := s - slast
				wbal := in - out - delsto
				if math.Abs(wbal) > nearzero {
					fmt.Printf("step: %d  cell ID: %d  Topm: %.5f  Qo: %.3f, freeboard: %.3f\n", i, c, p.gw[sid].M, Qo, freeboard)
					fmt.Printf("  in: %.5f = pre: %.5f + lag: %.5f\n", in, y, lag[c])
					fmt.Printf(" out: %.5f = aet: %.5f + netgwe: %.5f + genro: %.5f\n", out, a, g, r)
					fmt.Printf("  ds: %.5f = sto: %.5f - slats: %.5f\n", delsto, s, slast)
					fmt.Printf(" cell %d: pre-runoff water-balance error, |wbal(pre)| = %.5e m\n", c, math.Abs(wbal))
					if math.Abs(lag[c]) > 1e5 || math.Abs(r) > 1e5 {
						kill = true
					}
				}
				wbsum += wbal
			}()

			// cascade
			if b.ds[c] == -1 { // outlet cell
				if _, ok := p.gw[c]; !ok {
					fmt.Printf(" model error: outlet not assigned a groundwater reservoir")
				}
				p.gw[c].Dm -= gsumsws[sid] / gcntsws[sid] // add recharge [m/ts]
				rsum += r                                 //+ hbf*p.celr[c] // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case p.pa[c]=0.
				lag[c] = 0.
				gr[c] += r * 1000.
			} else {
				if _, ok := p.gw[c]; ok {
					p.gw[c].Dm -= gsumsws[sid] / gcntsws[sid] // add recharge [m/ts]
				}
				rt := r * p.p0[c]
				lag[c] = r * (1. - p.p0[c]) // retention
				if lag[c] > 1. {
					rt += lag[c] - 1.
					lag[c] = 1.
				}
				lag[b.ds[c]] += rt
				gr[c] += rt * 1000.
			}

			// // groundwater discharge
			// sid := b.rtr.sws[c]
			// di, hs := p.gw[sid].GetDi(c), 0.
			// // if v, ok := p.gw[sid].Qs[c]; ok {
			// // 	hb = v * math.Exp((Qo-di)/p.gw[sid].M) // lateral discharge to streams [m/ts]
			// // 	bfsum += hb
			// // }
			// // if c == 14153985 {
			// // 	fmt.Printf(":")
			// // }
			// g, x, gwe := 0., 0., p.ws[c].UpdatePercWT(di)
			// if gwe < 0 { // groundwater exchange (negative: recharge, positive: discharge)
			// 	g = -gwe
			// 	di = 0.
			// 	gsum += g
			// } else {
			// 	x = gwe                        // excess discharge to surface
			// 	hs = p.ws[c].Storage() - slast // seepage into soil zone
			// 	gsum -= x + hs
			// 	xsum += x
			// 	spsum += hs
			// }
			// r := p.ws[c].UpdateP(y + x + lag[c]) //+ hb) // runoff
			// a := p.ws[c].UpdateEp(ep)            // aet

			// if di < -freeboard { // saturation excess runoff (Di: groundwater deficit)
			// 	// if v, ok := p.gw[sid].Qs[c]; ok { // lateral discharge to streams [m]
			// 	// 	hb = v // capping at Di=0 as excess is being handled by the SMA
			// 	// }
			// 	di += freeboard
			// 	xsum -= di // saturation excess runoff
			// } else {
			// 	// if v, ok := p.gw[sid].Qs[c]; ok { // lateral discharge to streams [m]
			// 	// 	hb = v * math.Exp(-di/p.gw[sid].M)
			// 	// }
			// 	di = 0.
			// }

			// // update HRU
			// // a, r, g := p.ws[c].Update(y-di+lag[c]+hb, ep)
			// r := p.ws[c].UpdateP(y - di + lag[c] + hb) // runoff
			// g := 0.                                    // recharge
			// if di >= 0. {                              // only recharge when deficit is available; otherwise reject
			// 	g = p.ws[c].UpdatePerc()
			// }
			// a := p.ws[c].UpdateEp(ep) // aet
			// if a < 0. {
			// 	log.Fatalf(" hru water-balance error, HRU ET = %.3e mm\n", a*1000.)
			// }
			// if r < 0. {
			// 	log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm\n", r*1000.)
			// }
			// if g < 0. {
			// 	log.Fatalf(" hru water-balance error, HRU potential recharge = %.3e mm\n", g*1000.)
			// }
			// ysum += y // sum precipitation (rainfall + snowmelt)
			// asum += a
			// gsum += g - (x + hs)         //+ hb)         // sum recharge less discharge [m]
			// gsumsws[sid] += g - (x + hs) // + hb) // sum recharge less discharge [m], on a sws basis
			// gcntsws[sid]++               // count recharge
			// gp[c] += y * af              // sum grid precip monitor [mm/yr]
			// ga[c] += a * af              // sum grid AET [mm/yr]
			// gg[c] += (g - (x + hs)) * af // + hb)) * af // sum grid net groundwater exchange [mm/yr]

			// hb := 0.
			// if v, ok := p.gw[sid].Qs[c]; ok {
			// 	hb = v * math.Exp((Qo-di)/p.gw[sid].M) // lateral discharge to streams [m/ts]
			// 	bfsum += hb
			// }

		}

		// // groundwater discharge to streams
		// for s, gw := range p.gw {
		// 	hbsum := 0.
		// 	for c, v := range gw.Qs {
		// 		hb :=  v * math.Exp((Qo-gw.GetDi(c))/gw.M) // lateral discharge to streams [m/ts]
		// 		bfsum += hb	
		// 		hbsum += hb
		// 		if b.ds[c] == -1 { // outlet cell
		// 			rsum += hb
		// 		} else {
		// 			lag[b.ds[c]] += hb						
		// 		}
		// 		gr[c] += hb * 1000.
		// 	}
		// 	gw.Dm += hbsum/gcntsws[s]
		// }

		ysum /= b.fncid
		ssum /= b.fncid
		ssumlast /= b.fncid
		asum /= b.fncid
		rsum /= b.fncid
		xsum /= b.fncid
		gsum /= b.fncid
		bfsum /= b.fncid
		spsum /= b.fncid

		// lumped baseflow		 
		for s, gw := range p.gw {
			hb := Qo * math.Exp(-gw.Dm/gw.M)
			gw.Dm += hb
			bfsum += hb * p.swsr[s]
		}
		rsum += bfsum

		slag := 0.
		for k, v := range lag {
			slag += v
			gl[k] = v * 1000.
		}
		slag /= b.fncid
		slaglast /= b.fncid

		gwd := 0. // current basin groundwater deficit
		for k, v := range p.gw {
			gwd += v.Dm * p.swsr[k]
		}

		// wbsum /= b.fncid
		if math.Abs(wbsum) > nearzero {
			fmt.Printf(" step: %d  freeboard: %.5f\n", i, freeboard)
			fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" (integrated) hru water-balance error, |wbsum| = %.5e m\n", math.Abs(wbsum))
		}
		// wbalBasin := ysum + bfsum + xsum + ssumlast + slaglast - (ssum + asum + rsum + gsum + slag)

		// func() {
			gain, loss := ysum, asum+rsum
			s0, s1 := -gwdlast+ssumlast+slaglast, -gwd+ssum+slag
			wbalBasin := gain + s0 - (loss + s1)
			if math.Abs(wbalBasin) > nearzero {
				fmt.Printf(" step: %d  Qo: %.5f  freeboard: %.5f\n", i, Qo, freeboard)
				fmt.Printf(" pre: %.5f   ex: %.5f  lag: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, slag, asum, gsum, rsum, v[met.UnitDischarge])
				fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gwsto: %.5f  wbalBasin: % .10f\n", ssumlast, ssum, gwdlast, gwd, wbalBasin)
				fmt.Printf(" basin water-balance error, |wbalBasin| = %.5e m\n", math.Abs(wbalBasin))
				if math.Abs(gwd) > 1e5 {
					kill = true
				}
			}
		// }()

		if kill {
			break
		}

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * h2cms
		s[i] = rsum * h2cms
		bf[i] = bfsum * h2cms // groundwater discharge to streams [m³/s]
		xs[i] = xsum * h2cms   // groundwater discharge to surface [m³/s]
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
