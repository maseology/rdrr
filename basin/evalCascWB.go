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

// evalCascWB same as evalCasc() except with rigorous mass balance checking
func (b *subdomain) evalCascWB(p *sample, print bool) (of float64) {
	// constants
	nstep := b.frc.h.Nstep()                      // number of time steps
	dtb, dte, intvl := b.frc.h.BeginEndInterval() // start date, end date, time step interval [s]
	h2cms := b.contarea / float64(intvl)          // [m/ts] to [m³/s] conversion factor
	af := 365.24 * 1000. / float64(nstep)         // aggrate conversion factor [mm/yr]

	// monitors
	// outlet discharge [m³/s]: observes, simulated, baseflow
	o, s, bf, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	// water budget [mm]
	ws, wd, wa, wg, wx, wf, wk := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep)
	// distributed monitors [mm/yr]
	gp, ga, gr, gg := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid), make(map[int]float64, b.ncid)

	// printouts (deferred)
	defer func() {
		rmse := objfunc.RMSEi(o, s)
		of = rmse //(1. - kge) //* (1. - mwr2)
		if print {
			kge := objfunc.KGEi(o, s)
			mwr2 := objfunc.Krausei(computeMonthly(dt, o, s, float64(intvl), b.contarea))
			sumHydrograph(dt, o, s, bf)
			sumHydrographWB(dt, ws, wd, wa, wg, wx, wf, wk)
			sumMonthly(dt, o, s, float64(intvl), b.contarea)
			saveBinaryMap1(gp, "precipitation.rmap")
			saveBinaryMap1(ga, "aet.rmap")
			saveBinaryMap1(gr, "runoff.rmap")
			saveBinaryMap1(gg, "recharge.rmap")
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			fmt.Printf("  KGE: %.3f  NSE: %.3f  mon-wr2: %.3f  Bias: %.3f\n", kge, objfunc.NSEi(o, s), mwr2, objfunc.Biasi(o, s))
		}
	}()

	// initialize cell-based state variables; initialize monitors
	hq, klsum := make(map[int]float64, b.ncid), 0. // depth of mobile water [m]
	for _, c := range b.cids {
		hq[c] = 0.
		gp[c] = 0.
		ga[c] = 0.
		gr[c] = 0.
		gg[c] = 0.
	}

	// run model
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		fmt.Println(d)
		v := b.frc.c[d] // climate forcings

		// basin-HRU water budgeting [water balance; atmos.yeild; AET; runoff; excess runoff; GW infiltration; runon infiltration; storage at end, storage at beginning; mobile runoff at end; mobile runoff at beginning; baseflow]
		wbsum, ysum, asum, rsum, xsum, fdsum, fqsum, ssum, slsum, ksum, bfsum := 0., 0., 0., 0., 0., 0., 0., 0., 0., 0., 0.
		// for _, c := range b.cids {
		// 	klsum += hq[c]
		// }
		gwdlast, ggwsum, ggwcnt := 0., make(map[int]float64, len(p.gw)), make(map[int]float64, len(p.gw))
		for k, v := range p.gw {
			gwdlast += v.Dm * v.Ca
			ggwsum[k] = 0. // sum of recharge for gw res k
			ggwcnt[k] = 0. // count of recharge for gw res k
		}
		gwdlast /= b.contarea // basin groundwater deficit at beginning of timestep

		for _, c := range b.cids {
			y := v[met.AtmosphericYield]     // precipitation/atmospheric yield (rainfall + snowmelt)
			ep := v[met.AtmosphericDemand]   // evaporative demand
			ep *= b.strc.f[c][d.YearDay()-1] // adjust for slope-aspect

			slast := p.ws[c].Storage() // total HRU storage at beginning of timestep [m]
			slsum += slast
			fd, fq := 0., 0. // infiltration from groudwater, runon, respectively

			// check for consistent gw res mapping
			sid, ok := b.mpr.sws[c]
			if !ok && len(b.mpr.sws) > 0 {
				log.Fatalf(" evalCascKineWB sws error")
			}
			if _, ok := p.gw[sid]; !ok {
				log.Fatalf(" evalCascKineWB gw[sws] error")
			}

			// groundwater discharge
			di := p.gw[sid].GetDi(c) // groundwater deficit [m]
			ge := 0.                 // groundwater evaporation
			if di < 0. {             // groundater excess/discharge
				ggwsum[sid] += di // negative recharge (groundwater discharge) [m]
				if -di > ep {     // evaporate from groundwater
					ge = ep
				} else {
					ge = -di
				}
				di += ge
				ep -= ge // remaining evaporation demand

				fd = p.ws[c].Deficit() // available hru storage
				if fd < 0. {
					log.Fatalf(" hru water-balance error, HRU deficit less than zero: f = %.3e mm", fd*1000.)
				} else if fd > 0. { // available soil zone storage
					if -di > fd { // surplus
						di += fd
						rd := p.ws[c].UpdateStorage(fd) // add gw excess to storage
						if math.Abs(rd) > nearzero {
							log.Fatalf(" hru water-balance error, HRU infiltration from groundwater discharge exceeds capacity: f = %.3e mm, x = %.3e mm", fd*1000., rd*1000.)
						}
						if math.Abs(p.ws[c].Deficit()) > nearzero {
							log.Fatalf(" hru water-balance error, HRU infiltration from groundwater discharge failed to meet capacity: f = %.3e mm, deficit = %.3e mm", fd*1000., p.ws[c].Deficit()*1000.)
						}
						if di > nearzero {
							log.Fatalf(" hru water-balance error, infiltration exceeded available GW discharge = %.3e mm", -di*1000.)
						}
					} else {
						rd := p.ws[c].UpdateStorage(-di) // add gw excess to storage
						fd = -di                         // infiltration from GW to soil zone + surface storage
						di = 0.
						if math.Abs(rd) > nearzero {
							log.Fatalf(" hru water-balance error, HRU infiltration from groundwater discharge exceeds capacity: f = %.3e mm, x = %.3e mm", fd*1000., rd*1000.)
						}
					}
				}
				xsum -= di // saturation excess runoff = -di
			} else {
				di = 0. // setting deficits (di>0) to zero for computations below
			}

			// update HRU
			a, r, g := p.ws[c].Update(y, ep)
			if a < 0. {
				log.Fatalf(" hru water-balance error, HRU ET = %.3e mm", a*1000.)
			}
			if r < 0. || math.IsNaN(r) {
				log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm", r*1000.)
			}
			if g < 0. {
				log.Fatalf(" hru water-balance error, HRU potential recharge = %.3e mm", g*1000.)
			}
			asum += a + ge   // sum AET
			ggwsum[sid] += g // sum recharge
			ggwcnt[sid]++    // count recharge
			ep -= a
			if ep < -nearzero {
				log.Fatalf(" hru water-balance error, available ET < 0: ep = %.3e mm", ep*1000.)
			}

			// pre-runoff waterbalance
			s1 := p.ws[c].Storage()
			ds1 := s1 - slast // change in storage
			wbal1 := y + fd - (ds1 + r + g + a)
			if math.Abs(wbal1) > nearzero {
				fmt.Printf(" pre: %.5f   ex: %.5f genro: %.5f  aet: %.5f  rch: %.5f gw2sto: %.5f  sto: %.5f slast: %.5f\n", y, -di, r, a, g, fd, s1, slast)
				log.Fatalf(" cell %d: water-balance error (pre-runoff), |wbal| = %.5e m", c, math.Abs(wbal1))
			}
			ysum += y              // sum precipitation (rainfall + snowmelt)
			gp[c] += y * af        // sum grid precip monitor [mm/yr]
			ga[c] += (a + ge) * af // sum grid AET [mm/yr]
			gg[c] += (g + di) * af // sum grid recharge [mm/yr]; -di = groundwater discharge

			// Cascade
			rt := r - di // adding groundwater excess to precipitation excess (generated runoff)
			hqsv := hq[c]
			if rt > 0. {
				hq[c] += rt
			}
			fq = p.ws[c].Infiltrability() // potential infiltration from runoff stor
			if fq < 0. {
				log.Fatalf(" hru water-balance error, HRU potential infiltration = %.3e mm", fq*1000.)
			}
			if fq > hq[c] {
				fq = hq[c]
			}
			hq[c] -= fq
			rr := p.ws[c].UpdateStorage(fq) // add infiltration
			if math.Abs(rr) > nearzero {
				log.Fatalf(" hru water-balance error, HRU infiltration from runon exceeds capacity: f = %.3e mm, x = %.3e mm", fq*1000., rr*1000.)
			}
			hqo := p.p0[c] * hq[c] // outflow [m]
			hq[c] -= hqo
			if hqo < -nearzero {
				log.Fatalf(" mobile water-balance error, negative outflow computed = %.3e mm", hqo*1000.)
			}
			if b.ds[c] == -1 {
				rsum += hqo // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case sc[c]=0. [m/ts]
				if _, ok := p.gw[c]; !ok {
					log.Fatalf(" model error: outlet not assigned a groundwater reservoir")
				}
				bfsum += p.gw[c].Update(ggwsum[sid]/ggwcnt[sid]) * p.gw[c].Ca // basin baseflow [m³/ts]
			} else {
				if _, ok := p.gw[c]; ok {
					Qbf := p.gw[c].Update(ggwsum[sid] / ggwcnt[sid]) // baseflow from gw[c] discharging to cell c [m/ts]
					hq[b.ds[c]] += Qbf                               // adding baseflow to input of downstream cell [m/ts]
					bfsum += Qbf * p.gw[c].Ca                        // basin baseflow [m³/ts]
				}
				hq[b.ds[c]] += hqo
			}
			gr[c] += hqo * 1000.

			me := 0.                   // evaporation from mobile water
			if ep > 0. && hq[c] > 0. { // evaporate from mobile
				if ep > hq[c] {
					me = hq[c]
					ep -= hq[c]
					hq[c] = 0.
				} else {
					me = ep
					hq[c] -= ep
					ep = 0.
				}
				asum += me
			}

			// qosv := qo[c] // [m²/s]
			// if rt > 0. {
			// 	qo[c] = p.p0[c]*(qi[c]+gc*rt) + p.p1[c]*qo[c] // [m²/s]
			// } else {
			// 	fq = p.ws[c].Infiltrability() // potential infiltration from runoff stor
			// 	if fq < 0. {
			// 		log.Fatalf(" hru water-balance error, HRU potential infiltration = %.3e mm", fq*1000.)
			// 	}
			// 	fx := (p.p0[c]*qi[c] + p.p1[c]*qo[c]) / p.p0[c] / gc // max available from runoff stor to infiltrate [m]
			// 	if fx < nearzero {
			// 		fx = 0.
			// 	}
			// 	if fq > fx {
			// 		fq = fx
			// 	}
			// 	qo[c] = p.p0[c]*(qi[c]-gc*fq) + p.p1[c]*qo[c]
			// 	rr := p.ws[c].UpdateStorage(fq) // add infiltration
			// 	if math.Abs(rr) > nearzero {
			// 		log.Fatalf(" hru water-balance error, HRU infiltration from runon exceeds capacity: f = %.3e mm, x = %.3e mm", fq*1000., rr*1000.)
			// 	}
			// 	if qo[c] < -nearzero {
			// 		log.Fatalf(" hru water-balance error, negative runoff computed = %.3e mm", qo[c]/gc*1000.)
			// 	}
			// }
			// if b.ds[c] == -1 {
			// 	rsum += qo[c] / gc // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case sc[c]=0. [m/ts]
			// 	if _, ok := p.gw[c]; !ok {
			// 		log.Fatalf(" model error: outlet not assigned a groundwater reservoir")
			// 	}
			// 	bfsum += p.gw[c].Update(ggwsum[sid] / ggwcnt[sid]) // basin baseflow [m³/ts]
			// } else {
			// 	if _, ok := p.gw[c]; ok {
			// 		Qbf := p.gw[c].Update(ggwsum[sid] / ggwcnt[sid]) // baseflow from gw[c] discharging to cell c [m³/ts]
			// 		qi[b.ds[c]] += Qbf / b.strc.w                    // adding baseflow to input of downstream cell [m²/ts]
			// 		bfsum += Qbf                                     // basin baseflow [m³/ts]
			// 	}
			// 	qi[b.ds[c]] += qo[c]
			// }
			// gr[c] += qo[c] / gc * 1000.
			// ki, ko := qi[c]/gc, qo[c]/gc // runon; runoff
			// dk := (qi[c] - qo[c]) / gc

			// HRU waterbalance (post runoff)
			s2 := p.ws[c].Storage()
			ds2 := s2 - slast // change in storage
			wbal2 := y + fd + fq - (ds2 + r + g + a)
			if math.Abs(wbal2) > nearzero {
				fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f  slast: %.5f  aet: %.5f  rch: % .5f   ri: %.5f   ro: %.5f\n", y, -di, s2, slast, a, g, hq[c], hqo)
				fmt.Printf(" gw2sto: %.5f  ro2sto: %.5f\n", fd, fq)
				log.Fatalf(" cell %d: HRU water-balance error, |wbal| = %.5e m", c, math.Abs(wbal2))
			}

			// // check mobile runoff
			dsk := hq[c] - hqsv
			wbalM := dsk - (r - di - fq - me - hqo)
			if wbalM < -nearzero {
				fmt.Printf(" inflow: %f  genro: %f  excess: %f  infil: %f  evap: %f outflow: %f\n", hqsv, r, -di, fq, me, hqo)
				log.Fatalf(" mobile water balance error: negative net volume %.3e mm", wbalM*1000.)
			}
			// wbalM := dsk + qosv/gc
			// if wbalM < -nearzero || math.Log10(ki) >= 5. {
			// 	ucnt := b.strc.t.UpCnt(c)
			// 	fmt.Printf(" C0: %f  C2: %f\n", p.p0[c], p.p1[c])
			// 	fmt.Println(ucnt, rt, float64(ucnt)*rt)
			// 	// fmt.Println(b.ds[c], ggwsum[sid]/ggwcnt[sid], ggwcnt[sid])
			// 	fmt.Println(c, qi[c], qo[c], qosv, dk)
			// 	fmt.Printf(" inflow: %f  genro: %f  excess: %f  infil: %f  outflow: %f  outflow_prev: %f\n", ki, r, -di, fq, ko, qosv/gc)
			// 	fmt.Printf(" likely breaking of the Courant condition..\n")
			// 	log.Fatalf(" mobile water balance error: negative net volume %.3e mm", wbalM*1000.)
			// }

			// CE waterbalance
			dsg := g - fd - ge + di
			dsa := ds2 + dsk + dsg
			wbal3 := y - (dsa + hqo + (a + ge + me))
			if math.Abs(wbal3) > nearzero { //&& math.Log10(ki) < 5. {
				fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f  slast: %.5f  aet: %.5f  rch: % .5f   ri: %.5f   ro: %.5f\n", y, -di, s2, slast, (a + ge + me), g, hq[c], hqo)
				fmt.Printf(" gw2sto: %.5f  ro2sto: %.5f\n", fd, fq)
				log.Fatalf(" cell %d: CE water-balance error, |wbal| = %.5e m", c, math.Abs(wbal3))
			}

			wbsum += wbal2 // basin waterbalance sum
			ssum += s2     // total basin-HRU storage at end of timestep [m]
			fdsum += fd    // total HRU infiltration from groundwater
			fqsum += fq    // total HRU infiltration from runon
			ksum += hq[c]  // total volume of "mobile/active runoff"
			// hq[c] = 0.            // reset
		}

		// normalize wbsum, asum, rsum, xsum, fdsum, fqsum, ssum, slsum, ksum, bfsum
		wbsum /= b.fncid
		ysum /= b.fncid  // precipitation (rainfall + snowmelt)
		asum /= b.fncid  // evaporation
		rsum /= b.fncid  // runoff (at outlet)
		xsum /= b.fncid  // saturation excess runoff
		fdsum /= b.fncid // infiltration from GW to storage
		fqsum /= b.fncid // infiltration from RO to storage
		ssum /= b.fncid  // current storage
		slsum /= b.fncid // last storage
		ksum /= b.fncid  // volume of mobile water / "mobile" storage
		gsum, gwcnttot := 0., 0.
		for i, v := range ggwsum {
			gsum += v / ggwcnt[i]
			gwcnttot += ggwcnt[i] * 50. * 50.
		}
		gsum /= b.fncid     // recharge
		bfsum /= b.contarea // unit baseflow ([m³/ts] to [m/ts])
		rsum += bfsum

		if math.Abs(wbsum) > nearzero {
			fmt.Printf(" step: %d\n", i)
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, asum, gsum, rsum, v[met.UnitDischarge])
			log.Fatalf(" (integrated) hru water-balance error, |wbsum| = %.5e m", math.Abs(wbsum))
		}

		gwd := 0.
		for _, v := range p.gw {
			gwd += v.Dm * v.Ca
		}
		gwd /= b.contarea // current basin groundwater deficit

		dgwd := -(gwd - gwdlast) // negative change in deficit = gains to gw storage
		dsh := ssum - slsum
		dsk := ksum - klsum
		wbalBasin := ysum - (dsh + dgwd + dsk + asum + rsum)
		if math.IsNaN(wbalBasin) {
			log.Fatalf(" basin water-balance error, NaN")
		}
		if math.Abs(wbalBasin) > nearzero { //&& math.Log10(gwd) < 5. {
			fmt.Printf(" step: %d\n", i)
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", ysum, xsum, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gwsto: %.5f  klast: %.5f  ksto: %.5f\n", slsum, ssum, gwdlast, gwd, klsum, ksum)
			fmt.Printf(" dsto-hru: %.5f  dsto-gw: %.5f  dsto-k: %.5f  wbal: % .5e\n", dsh, dgwd, dsk, wbalBasin)
			fmt.Printf(" basin water-balance error, |wbalBasin| = %.5e m", math.Abs(wbalBasin))
		}

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * h2cms // observed discharge [m³/s]
		bf[i] = bfsum * h2cms               // groundwater discharge to streams [m³/s]
		s[i] = rsum * h2cms                 // total runoff at outlet [m³/s]
		ws[i] = ssum * 1000.                // CE storage
		wd[i] = gwd * 1000.                 // GW deficit
		wg[i] = gsum * 1000.                // groundwater recharge
		wx[i] = xsum * 1000.                // saturation excess runoff
		wk[i] = ksum * 1000.                // mobile runoff
		wa[i] = asum * 1000.                // evaporation
		wf[i] = (fdsum + fqsum) * 1000.     // infiltration
		i++
		klsum = ksum
	}
	return
}
