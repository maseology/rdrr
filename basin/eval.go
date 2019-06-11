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

// evalCascKineWB same as evalCascKine() except with rigorous mass balance checking
func (b *subdomain) evalCascKineWB(p *sample, print bool) (of float64) {
	// constants
	nstep := b.frc.h.Nstep()
	dtb, dte, intvl := b.frc.h.BeginEndInterval()
	gc := b.strc.w / float64(intvl)        // grid celerity (w/ts)
	cf := b.contarea / float64(intvl)      // q to cms conversion factor
	af := 365.25 * 1000.0 / float64(nstep) // aggrate conversion factor (mm/yr)

	// timeseries monitors
	o, s, g, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	ws, wa, wg, wx, wf := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep)

	// distributed monitors
	gg, gr := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid)

	// printouts (deferred)
	defer func() {
		kge := objfunc.KGEi(o, s)
		mwr2 := objfunc.Krausei(computeMonthly(dt, o, s, float64(intvl), b.contarea))
		of = (1. - kge) //* (1. - mwr2)
		if print {
			sumHydrograph(dt, o, s, g)
			sumHydrographWB(dt, ws, wa, wg, wx, wf)
			sumMonthly(dt, o, s, float64(intvl), b.contarea)
			saveBinaryMap1(gr, "runoff.rmap")
			saveBinaryMap1(gg, "recharge.rmap")
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			fmt.Printf("  KGE: %.3f  NSE: %.3f  mon-wr2: %.3f  Bias: %.3f\n", kge, objfunc.NSEi(o, s), mwr2, objfunc.Biasi(o, s))
		}
	}()

	// initialize cell-based variables
	qi, qo := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid) // inflow this timestep, outflow last timestep
	for _, c := range b.cids {
		qi[c] = 0.
		qo[c] = 0.
		gr[c] = 0.
		gg[c] = 0.
	}

	// run model
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		// fmt.Println(d)
		v := b.frc.c[d]
		gwlast := p.gw.Dm
		wbsum, asum, rsum, xsum, gsum, fdsum, frsum, ssum, slsum, ksum := 0., 0., 0., 0., 0., 0., 0., 0., 0., 0.
		for _, c := range b.cids {

			// if c == 148029 || c == 131656 || c == 130145 || c == 141557 || c == 141053 || c == 132642 || c == 137108 {
			// 	println("pop")
			// }

			slast := p.ws[c].Storage() // total HRU storage at beginning on timestep
			slsum += slast
			fd, fr := 0., 0. // infiltration from groudwater, runon, respectively

			di := p.gw.GetDi(c) // groundwater deficit
			if di < 0. {        // groundater excess/discharge
				gsum += di             // negative recharge
				fd = p.ws[c].Deficit() // available soilzone storage
				if fd < 0. {
					log.Fatalf(" hru water-balance error, potential HRU infiltration less than zero: f = %.3e mm", fd*1000.)
				} else if fd > 0. { // available soil zone storage
					if -di > fd {
						di += fd
						rd := p.ws[c].UpdateStorage(fd) // add gw excess to storage
						if math.Abs(rd) > nearzero {
							log.Fatalf(" hru water-balance error, HRU infiltration from groundwater discharge exceeds capacity: f = %.3e mm, x = %.3e mm", fd*1000., rd*1000.)
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
				di = 0.
			}

			// add precipitation (rainfall + snowmelt)
			a, r, g := p.ws[c].Update(v[met.AtmosphericYield], v[met.AtmosphericDemand]*b.strc.f[c][d.YearDay()-1])
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
			gg[c] += g * af

			// pre-runoff waterbalance
			s1 := p.ws[c].Storage()
			ds1 := s1 - slast
			wbal1 := v[met.AtmosphericYield] + fd - (ds1 + r + g + a)
			if math.Abs(wbal1) > nearzero {
				fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f  slast: %.5f  aet: %.5f  rch: %.5f gw2sto: %.5f\n", v[met.AtmosphericYield], -di, s1, slast, a, g, fd)
				log.Fatalf(" cell %d: water-balance error (pre-runoff), |wbal| = %.5e m", c, math.Abs(wbal1))
			}
			r -= di // adding groundwater excess to precipitation excess

			// kinematic cascade
			if r > 0 {
				qo[c] = p.p0[c]*qi[c] + p.p1[c]*(qo[c]+gc*r)
			} else {
				fr = p.ws[c].Infiltrability() // potential infiltration
				if fr < 0. {
					log.Fatalf(" hru water-balance error, HRU potential infiltration = %.3e mm", fr*1000.)
				}
				fx := (p.p0[c]*qi[c] + p.p1[c]*qo[c]) / p.p1[c] / gc // max available to infiltrate
				if fx < nearzero {
					fx = 0.
				}
				if fr > fx {
					fr = fx
				}
				qo[c] = p.p0[c]*qi[c] + p.p1[c]*(qo[c]-gc*fr)
				rr := p.ws[c].UpdateStorage(fr) // add infiltration
				if math.Abs(rr) > nearzero {
					log.Fatalf(" hru water-balance error, HRU infiltration from runon exceeds capacity: f = %.3e mm, x = %.3e mm", fr*1000., rr*1000.)
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
			gr[c] += qo[c] * af

			// waterbalance
			s := p.ws[c].Storage()
			ki, ko := qi[c]/gc, qo[c]/gc
			ds := s - slast
			wbal := v[met.AtmosphericYield] - di + fd + fr - (ds + r + g + a)
			if math.Abs(wbal) > nearzero {
				fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f  slast: %.5f  aet: %.5f  rch: % .5f   ri: %.5f   ro: %.5f\n", v[met.AtmosphericYield], -di, s, slast, a, g, ki, ko)
				fmt.Printf(" gw2sto: %.5f  ro2sto: %.5f\n", fd, fr)
				log.Fatalf(" cell %d: water-balance error, |wbal| = %.5e m", c, math.Abs(wbal))
			}
			wbsum += wbal
			ssum += s
			fdsum += fd
			frsum += fr
			ksum += ki + r - ko
			qi[c] = 0.
		}
		ssum /= b.fncid  // current storage
		slsum /= b.fncid // last storage
		asum /= b.fncid  // evaporation
		rsum /= b.fncid  // runoff (at outlet)
		xsum /= b.fncid  // saturation excess runoff
		gsum /= b.fncid  // recharge
		fdsum /= b.fncid // infiltration from GW to storage
		frsum /= b.fncid // infiltration from RO to storage
		ksum /= b.fncid  // volume of mobile water / "mobile" storage

		bf := p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])
		rsum += bf

		wbsum /= b.fncid
		if math.Abs(wbsum) > nearzero {
			fmt.Printf(" step: %d  TOPMODEL m: %.5f\n", i, p.gw.M)
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
			log.Fatalf(" (integrated) hru water-balance error, |wbsum| = %.5e m", math.Abs(wbsum))
		}
		wbalBasin := v[met.AtmosphericYield] - gwlast + slsum + frsum - (-p.gw.Dm + ssum + asum + rsum + ksum)
		if math.Abs(wbalBasin) > nearzero && math.Log10(p.gw.Dm) < 5. {
			fmt.Printf(" step: %d  TOPMODEL m: %.5f\n", i, p.gw.M)
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gwsto: %.5f  wbal: % .5e\n", slsum, ssum, gwlast, p.gw.Dm, wbalBasin)
			log.Fatalf(" basin water-balance error, |wbalBasin| = %.5e m", math.Abs(wbalBasin))
		}

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * cf
		g[i] = bf * cf   // groundwater discharge to streams
		s[i] = rsum * cf // total runoff at outlet

		wg[i] = gsum * 1000.            // groundwater recharge
		wx[i] = xsum * 1000.            // saturation excess runoff
		ws[i] = ssum * 1000.            // CE storage
		wa[i] = asum * 1000.            // evaporation
		wf[i] = (fdsum + frsum) * 1000. // infiltration

		i++
	}
	return
}

// // evalCascKine evaluates (runs) the basin model with kinematic routing
// func (b *subdomain) evalCascKine(p *sample, print bool) (of float64) {
// 	nstep := b.frc.h.Nstep()
// 	dtb, dte, intvl := b.frc.h.BeginEndInterval()
// 	gc := b.strc.w / float64(intvl)   // grid celerity (w/ts)
// 	cf := b.contarea / float64(intvl) // q to cms conversion factor
// 	o, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
// 	defer func() {
// 		kge := objfunc.KGEi(o, s)
// 		mwr2 := objfunc.Krausei(computeMonthly(dt, o, s, float64(intvl), b.contarea))
// 		of = (1. - kge) * (1. - mwr2)
// 	}()

// 	// initialize
// 	qi, qo := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid) // inflow this timestep, outflow last timestep
// 	for _, c := range b.cids {
// 		qi[c] = 0.
// 		qo[c] = 0.
// 	}

// 	// run model
// 	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
// 		v := b.frc.c[d]
// 		rsum, gsum := 0., 0.
// 		for _, c := range b.cids {
// 			di := p.gw.GetDi(c)
// 			if di < -p.rill { // saturation excess runoff (Di: groundwater deficit)
// 				di += p.rill
// 				gsum += di // negative recharge
// 			} else {
// 				di = 0.
// 			}
// 			_, r, _ := p.ws[c].Update(v[met.AtmosphericYield]-di, v[met.AtmosphericDemand]*b.strc.f[c][d.YearDay()-1])

// 			// cascade
// 			d := 0.
// 			if r > 0 {
// 				qo[c] = p.p0[c]*qi[c] + p.p1[c]*(qo[c]+gc*r)
// 			} else {
// 				d = p.ws[c].Infiltrability()
// 				f := d                                               // potential infiltration
// 				fx := (p.p0[c]*qi[c] + p.p1[c]*qo[c]) / p.p1[c] / gc // max available to infiltrate
// 				if fx < nearzero {
// 					fx = 0.
// 				}
// 				if f > fx {
// 					f = fx
// 				}
// 				qo[c] = p.p0[c]*qi[c] + p.p1[c]*(qo[c]-gc*f)
// 				p.ws[c].UpdateStorage(f) // add infiltration
// 			}
// 			if b.ds[c] == -1 {
// 				rsum += qo[c] / gc // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case sc[c]=0.
// 			} else {
// 				qi[b.ds[c]] += qo[c]
// 			}

// 			qi[c] = 0.
// 		}
// 		rsum /= b.fncid

// 		bf := p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])
// 		rsum += bf

// 		// save results
// 		dt[i] = d
// 		o[i] = v[met.UnitDischarge] * cf
// 		s[i] = rsum * cf
// 		i++
// 	}
// 	return
// }

// // evalCascKineWB same as evalCascKine() except with rigorous mass balance checking
// func (b *subdomain) evalCascKineWB(p *sample, print bool) (of float64) {
// 	nstep := b.frc.h.Nstep()
// 	dtb, dte, intvl := b.frc.h.BeginEndInterval()
// 	gc := b.strc.w / float64(intvl)        // grid celerity (w/ts)
// 	cf := b.contarea / float64(intvl)      // q to cms conversion factor
// 	af := 365.25 * 1000.0 / float64(nstep) // aggrate conversion factor (mm/yr)

// 	// timeseries
// 	o, s, g, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
// 	ws, wa, wg, wx, wf := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep)

// 	// distributed
// 	gg, gr := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid)

// 	defer func() {
// 		kge := objfunc.KGEi(o, s)
// 		mwr2 := objfunc.Krausei(computeMonthly(dt, o, s, float64(intvl), b.contarea))
// 		of = (1. - kge) //* (1. - mwr2)
// 		if print {
// 			sumHydrograph(dt, o, s, g)
// 			sumHydrographWB(dt, ws, wa, wg, wx, wf)
// 			sumMonthly(dt, o, s, float64(intvl), b.contarea)
// 			saveBinaryMap1(gr, "runoff.rmap")
// 			saveBinaryMap1(gg, "recharge.rmap")
// 			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
// 			fmt.Printf("  KGE: %.3f  NSE: %.3f  mon-wr2: %.3f  Bias: %.3f\n", kge, objfunc.NSEi(o, s), mwr2, objfunc.Biasi(o, s))
// 		}
// 	}()

// 	// initialize cell-based variables
// 	qi, qo := make(map[int]float64, b.ncid), make(map[int]float64, b.ncid) // inflow this timestep, outflow last timestep
// 	for _, c := range b.cids {
// 		qi[c] = 0.
// 		qo[c] = 0.
// 		gr[c] = 0.
// 		gg[c] = 0.
// 	}

// 	// run model
// 	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
// 		// fmt.Println(d)
// 		v := b.frc.c[d]
// 		gwlast := p.gw.Dm
// 		wbsum, asum, rsum, xsum, gsum, fsum, ssum, slsum, ksum := 0., 0., 0., 0., 0., 0., 0., 0., 0.
// 		for _, c := range b.cids {
// 			slast := p.ws[c].Storage() // total HRU storage at beginning on timestep
// 			slsum += slast

// 			// groundater discharge to surface
// 			di := p.gw.GetDi(c) // saturation excess runoff = -di
// 			if di < -p.rill {   // saturation excess runoff above rill storage (Di: groundwater deficit)
// 				di += p.rill
// 				xsum -= di
// 				gsum += di // negative recharge
// 			} else {
// 				di = 0.
// 			}
// 			a, r, g := p.ws[c].Update(v[met.AtmosphericYield]-di, v[met.AtmosphericDemand]*b.strc.f[c][d.YearDay()-1])
// 			if a < 0. {
// 				log.Fatalf(" hru water-balance error, HRU ET = %.3e mm", a*1000.)
// 			}
// 			if r < 0. || math.IsNaN(r) {
// 				log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm", r*1000.)
// 			}
// 			if g < 0. {
// 				log.Fatalf(" hru water-balance error, HRU potential recharge = %.3e mm", g*1000.)
// 			}
// 			asum += a
// 			gsum += g
// 			gg[c] += g * af

// 			// cascade
// 			f := 0.
// 			if r > 0 {
// 				qo[c] = p.p0[c]*qi[c] + p.p1[c]*(qo[c]+gc*r)
// 			} else {
// 				f = p.ws[c].Infiltrability() // potential infiltration
// 				if f < 0. {
// 					log.Fatalf(" hru water-balance error, HRU potential infiltration = %.3e mm", f*1000.)
// 				}
// 				fx := (p.p0[c]*qi[c] + p.p1[c]*qo[c]) / p.p1[c] / gc // max available to infiltrate
// 				if fx < nearzero {
// 					fx = 0.
// 				}
// 				if f > fx {
// 					f = fx
// 				}
// 				qo[c] = p.p0[c]*qi[c] + p.p1[c]*(qo[c]-gc*f)
// 				r2 := p.ws[c].UpdateStorage(f) // add infiltration
// 				if math.Abs(r2) > nearzero {
// 					log.Fatalf(" hru water-balance error, HRU infiltration from runon exceeds capacity: f = %.3e mm, x = %.3e mm", f*1000., r2*1000.)
// 				}
// 				if qo[c] < -nearzero {
// 					log.Fatalf(" hru water-balance error, negative runoff computed = %.3e mm", qo[c]/gc*1000.)
// 				}
// 			}
// 			if b.ds[c] == -1 {
// 				rsum += qo[c] / gc // forcing outflow cells to become outlets simplifies proceedure, ie, no if-statement in case sc[c]=0.
// 			} else {
// 				qi[b.ds[c]] += qo[c]
// 			}
// 			gr[c] += qo[c] * af

// 			// waterbalance
// 			s := p.ws[c].Storage()
// 			ki, ko := qi[c]/gc, qo[c]/gc
// 			ds := s - slast
// 			wbal := v[met.AtmosphericYield] - di + f - (ds + r + g + a)
// 			if math.Abs(wbal) > nearzero {
// 				fmt.Printf(" pre: %.5f   ex: %.5f  sto: %.5f  slast: %.5f  aet: %.5f  rch: % .5f   ri: %.5f   ro: %.5f\n", v[met.AtmosphericYield], -di, s, slast, a, g, qi[c]/gc, qo[c]/gc)
// 				log.Fatalf(" cell %d: water-balance error, |wbal| = %.5e m", c, math.Abs(wbal))
// 			}
// 			wbsum += wbal
// 			ssum += s
// 			fsum += f
// 			ksum += ki + r - ko
// 			qi[c] = 0.
// 		}
// 		ssum /= b.fncid  // current storage
// 		slsum /= b.fncid // last storage
// 		asum /= b.fncid  // evaporation
// 		rsum /= b.fncid  // runoff (at outlet)
// 		xsum /= b.fncid  // saturation excess runoff
// 		gsum /= b.fncid  // recharge
// 		fsum /= b.fncid  // infiltration
// 		ksum /= b.fncid  // volume of mobile water / "mobile" storage

// 		bf := p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])
// 		rsum += bf

// 		wbsum /= b.fncid
// 		if math.Abs(wbsum) > nearzero {
// 			fmt.Printf(" step: %d  rillsto: %.5f  m: %.5f\n", i, p.rill, p.gw.M)
// 			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
// 			log.Fatalf(" (integrated) hru water-balance error, |wbsum| = %.5e m", math.Abs(wbsum))
// 		}
// 		wbalBasin := v[met.AtmosphericYield] - gwlast + slsum - (-p.gw.Dm + ssum + asum + rsum + ksum - fsum)
// 		if math.Abs(wbalBasin) > nearzero && math.Log10(p.gw.Dm) < 5. {
// 			fmt.Printf(" step: %d  rillsto: %.5f  m: %.5f\n", i, p.rill, p.gw.M)
// 			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
// 			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gwsto: %.5f  wbal: % .5e\n", slsum, ssum, gwlast, p.gw.Dm, wbalBasin)
// 			log.Fatalf(" basin water-balance error, |wbalBasin| = %.5e m", math.Abs(wbalBasin))
// 		}

// 		// save results
// 		dt[i] = d
// 		o[i] = v[met.UnitDischarge] * cf
// 		g[i] = bf * cf   // groundwater discharge to streams
// 		s[i] = rsum * cf // total runoff at outlet

// 		wg[i] = gsum * 1000. // groundwater recharge
// 		wx[i] = xsum * 1000. // saturation excess runoff
// 		ws[i] = ssum * 1000. // CE storage
// 		wa[i] = asum * 1000. // evaporation
// 		wf[i] = fsum * 1000. // infiltration

// 		i++
// 	}
// 	return
// }
