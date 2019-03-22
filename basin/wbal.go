package basin

// // RunWBAL basin model
// func (b *Basin) RunWBAL() (of float64) {
// 	nstep := b.fhd.Nstep()
// 	o, g, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
// 	defer func() {
// 		of = objfunc.KGEi(o, s)
// 		// C:/Users/mason/OneDrive/R/dygraph/obssim_csv_viewer.R
// 		mmio.WriteCSV("hydrograph.csv", "date,obs,excess,gw", dt, o, s, g)
// 		// mmio.ObsSim("hydrograph.png", o[730:], s[730:])
// 		fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
// 		fmt.Printf("  KGE: %.3f  Bias: %.3f\n", of, objfunc.Biasi(o, s))
// 	}()

// 	// run model
// 	dtb, dte, intvl := b.fhd.BeginEndInterval()
// 	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
// 		// fmt.Println(d)
// 		v := b.frc[d]
// 		// rsum, gsum := 0., 0.
// 		gwlast := b.gw.Dm
// 		wbal, asum, rsum, xsum, gsum, ssum, slsum := 0., 0., 0., 0., 0., 0., 0.
// 		for _, c := range b.cids {
// 			slast := b.bsn[c].Storage() // initial HRU storage
// 			slsum += slast
// 			di := b.gw.GetDi(c)
// 			if di < -b.rill { // saturation excess runoff (Di: groundwater deficit)
// 				di += b.rill
// 				xsum -= di // saturation excess runoff
// 				gsum += di // negative recharge
// 			} else {
// 				di = 0.
// 			}
// 			a, r, g := b.bsn[c].Update(v[met.AtmosphericYield]-di, v[met.AtmosphericDemand]*b.sif[c][d.YearDay()-1])
// 			if a < 0. {
// 				log.Fatalf(" hru water-balance error, HRU ET = %.3e mm", a*1000.)
// 			}
// 			if r < 0. {
// 				log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm", r*1000.)
// 			}
// 			if g < 0. {
// 				log.Fatalf(" hru water-balance error, HRU potential recharge = %.3e mm", g*1000.)
// 			}
// 			s := b.bsn[c].Storage()
// 			wbal += v[met.AtmosphericYield] - di + slast - (s + g + a + r)
// 			ssum += s
// 			asum += a
// 			rsum += r
// 			// di := b.gw.GetDi(c)
// 			// fct := 1.
// 			// if g > di { // saturation excess runoff (Di: groundwater deficit)
// 			// 	ex := g - di/fct
// 			// 	xsum += ex   // rejected recharge
// 			// 	rsum += ex   // rejected recharge
// 			// 	g = di / fct // negative recharge
// 			// }
// 			gsum += g
// 		}
// 		ssum /= b.fncid
// 		slsum /= b.fncid
// 		asum /= b.fncid
// 		rsum /= b.fncid
// 		xsum /= b.fncid
// 		gsum /= b.fncid
// 		bf := b.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])
// 		rsum += bf

// 		// fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
// 		if math.Abs(wbal/b.fncid) > 1e-10 {
// 			log.Fatalf(" hru water-balance error, |wbal| = %.3e mm", math.Abs(wbal)*1000.)
// 		}
// 		wbalBasin := v[met.AtmosphericYield] - gwlast + slsum - (-b.gw.Dm + ssum + asum + rsum)
// 		// fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gw: %.5f  wbal: % .2e\n", slsum, ssum, gwlast, b.gw.Dm, wbalBasin)
// 		if math.Abs(wbalBasin) > 1e-10 {
// 			log.Fatalf(" basin water-balance error, |wbalBasin| = %.3e mm", math.Abs(wbalBasin)*1000.)
// 		}

// 		// save results
// 		dt[i] = d
// 		o[i] = v[met.UnitDischarge] * b.contarea / 86400.0 // cms
// 		g[i] = bf * b.contarea / 86400.0
// 		s[i] = xsum * b.contarea / 86400.0 //bf //rsum
// 		i++
// 	}
// 	return
// }
