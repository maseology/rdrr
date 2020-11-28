package prep

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/maseology/glbopt"
	"github.com/maseology/goHydro/pet"
	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

const (
	parseRainMeltRule0thresh = 10.
	parseRainMeltRule1thresh = 0.005
	parseRainMeltRule2thresh = 0.001

	penmanWFa, penmanWFb = 0.009, 0.26 // calibrated ep parameters
)

// BuildFORC builds the gob containing forcing data.
// (1) loads FEWS NetCDF (bin) output
// (2) returns sorted dates
// (2) computes basin
// (3) parses precipitation into rainfall by optimizing t_crit
func BuildFORC(gobDir, ncfp string, cells []Cell, dtb, dte time.Time) *model.FORC {
	smid := make(map[int]bool)
	for _, c := range cells {
		if _, ok := smid[c.Mid]; !ok {
			smid[c.Mid] = true
		}
	}

	dts, ys, eao, mxr, _ := collectMeteoData(ncfp, smid, dtb, dte)

	cmxr := make(map[int]int, len(cells))
	for _, c := range cells {
		cmxr[c.Cid] = mxr[c.Mid]
	}

	frc := model.FORC{
		T:           dts,
		D:           [][][]float64{ys, eao},
		XR:          cmxr,
		IntervalSec: 86400 / 4,
	}

	if err := frc.SaveGob(gobDir + "FORC.gob"); err != nil {
		log.Fatalf(" BuildFORC error: %v", err)
	}

	return &frc
}

func collectMeteoData(ncfp string, smid map[int]bool, dtb, dte time.Time) (dts []time.Time, y, ea [][]float64, xr map[int]int, ndat int) {
	fmt.Println(" loading data exported from FEWS:", ncfp, "...")

	ndat, nsta, nvar, dat := loadNC(ncfp, smid) //[station_id][datetime][ precipitation_amount, surface_snow_and_ice_melt_flux, air_temperature, air_pressure, relative_humidity, wind_speed ]
	fmt.Printf("  %6d stations\n %6d timesteps\n %6d variables\n\n parsing precipitation form..\n", nsta, ndat, nvar)

	// yield, temperature and pressure is acquired on a sws basis
	y = make([][]float64, nsta)
	ea = make([][]float64, nsta)
	xr = make(map[int]int, nsta)
	k := 0

	for sid, d := range dat {
		if dts == nil {
			dts = sequentialDates(d, dtb, dte)
		}
		prec, smlt, temp, eao := make([]float64, len(dts)), make([]float64, len(dts)), make([]float64, len(dts)), make([]float64, len(dts))
		tl, pl, rhl := 10., 101.3, .85

		for i, dt := range dts {
			if _, ok := d[dt]; !ok {
				log.Fatalf("Date %v not found in %s\n", dt, ncfp)
			}
			vs := d[dt]
			prec[i] = float64(vs[0]) / 1000. // [m]
			smlt[i] = float64(vs[1]) / 1000. // [m/d]
			temp[i] = float64(vs[2])
			// pres[i] = float64(vs[3])
			// rhfc[i] = float64(vs[4])
			// wvel[i] = float64(vs[5])
			// visi[i] = float64(vs[6])
			p, rh, u := float64(vs[3]), float64(vs[4]), float64(vs[5])

			if math.IsNaN(prec[i]) {
				prec[i] = 0.
			}
			if math.IsNaN(smlt[i]) {
				smlt[i] = 0.
			}
			if math.IsNaN(temp[i]) {
				temp[i] = tl
			}
			tl = temp[i]
			if math.IsNaN(p) {
				p = pl
			}
			pl = p
			if math.IsNaN(rh) {
				rh = rhl
			}
			rhl = rh
			if math.IsNaN(u) {
				u = 0.
			}

			eaoo := pet.PenmanWind(temp[i], p, rh, u, penmanWFa, penmanWFb) * 60. * 60. * 6. // [m/6hr]
			if math.IsNaN(eaoo) {
				log.Fatalf("eaoo error 1 NaN")
			} else if eaoo < 0. || eaoo*86.4*365.24 > 2000. {
				log.Fatalf("eaoo error 2: %f", eaoo*86.4*365.24)
			}

			eao[i] = eaoo
		}

		xr[sid] = k
		y[k] = getAtomsYieldByStation(sid, dts, prec, smlt, temp) // expensive
		ea[k] = eao
		k++
	}
	// func() { // print summary
	// 	revxr, _ := mmio.InvertMap(xr)
	// 	for k, yy := range y {
	// 		ss, ee, f := 0., 0., 4.*365.24*1000./float64(len(yy))
	// 		for kk, v := range yy {
	// 			ss += v
	// 			ee += ea[k][kk]
	// 		}
	// 		if len(revxr[k]) != 1 {
	// 			log.Fatalln(" collectMeteoData print summary error")
	// 		}
	// 		fmt.Printf("%d: mid: %d  sy: %.1f  se: %.1f\n", k, revxr[k][0], ss*f, ee*f) // mm/yr
	// 	}
	// }()
	return
}

func loadNC(ncfp string, smid map[int]bool) (int, int, int, map[int]map[time.Time][]float32) {
	switch ext := mmio.GetExtension(ncfp); ext {
	case ".bin": // created using /@dev/python/src/FEWS/netcdf/ncToMet.py
		b := mmio.OpenBinary(ncfp)
		nt := int(mmio.ReadInt32(b))
		times := make([]time.Time, nt)
		for i := 0; i < nt; i++ {
			times[i] = time.Unix(mmio.ReadInt64(b), 0).UTC()
		}

		nsta := int(mmio.ReadInt32(b))
		nvar := int(mmio.ReadInt32(b))
		stacnt := 0

		dcol := make(map[int]map[time.Time][]float32, nsta)
		for i := 0; i < nsta; i++ {
			d := make(map[time.Time][]float32, nt)
			mid := int(mmio.ReadInt32(b))
			for j := 0; j < nt; j++ {
				a := make([]float32, nvar)
				for iv := 0; iv < nvar; iv++ {
					a[iv] = mmio.ReadFloat32(b)
				}
				d[times[j]] = a
			}
			if _, ok := smid[mid]; ok {
				dcol[mid] = d
				stacnt++
			}
		}

		if stacnt != nsta {
			dcol2 := make(map[int]map[time.Time][]float32, stacnt)
			for k, v := range dcol {
				dcol2[k] = v
			}
			return nt, stacnt, nvar, dcol2
		}
		return nt, nsta, nvar, dcol
	case ".nc":
		log.Fatalln("ERROR: current not supporting NetCDF files. Convert to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py.")
	default:
		log.Fatalln("ERROR: loadNC unsupported file type:" + ext)
	}
	return -1, -1, -1, nil
}

func sequentialDates(d map[time.Time][]float32, dtb, dte time.Time) []time.Time {
	var dts []time.Time
	for k := range d {
		if k.Before(dtb) {
			continue
		}
		if k.After(dte) {
			continue
		}
		dts = append(dts, k)
	}
	sort.Slice(dts, func(i, j int) bool { return dts[i].Before(dts[j]) })
	return dts
}

func getAtomsYieldByStation(sid int, dts []time.Time, p, m, t []float64) (y []float64) {
	// critical temperature optimization
	var sp, sm, sr float64
	trans := func(u float64) float64 { return mmaths.LinearTransform(-50., 10., u) }
	solv := func(u []float64) float64 { // solve critical temperature
		tc := trans(u[0])
		_, sp, sm, sr = parseRainMelt(dts, p, m, t, tc)
		spp := sm + sr
		return math.Abs(spp-sp) / sp
	}
	uFib, _ := glbopt.Fibonacci(solv)
	tc := trans(uFib)                               // critical temperature
	y, sp, sm, sr = parseRainMelt(dts, p, m, t, tc) // final run

	spp := sm + sr
	dd := float64(len(dts)) / 365.24 / 4. / 1000.
	fmt.Printf("%10d: %10.1f %10.1f (m%.1f r%.1f) %20.5f %20.5f\n", sid, sp/dd, spp/dd, sm/dd, sr/dd, (spp-sp)/sp, tc) // mm/yr

	func() { // check
		s := 0.
		for _, v := range y {
			s += v
		}
		if math.Abs(s-spp)/spp > 0.00001 {
			log.Fatalf(" parseRainMelt accumulation error sum(y):%.1f  sp:%.1f  spp:%.1f  sr:%.1f  sm:%.1f", s/dd, sp/dd, spp/dd, sr/dd, sm/dd)
		}
	}()

	return
}

func parseRainMelt(orderedDatetimeList []time.Time, precip, snowmelt, temp []float64, tc float64) ([]float64, float64, float64, float64) {
	y := make([]float64, len(precip))
	sp, sm, smk, sr := 0., 0., 0., 0.
	var smDist map[time.Time]float64
	for k, dt := range orderedDatetimeList {
		yy := 0.
		sp += precip[k]
		if temp[k] >= tc {
			yy = precip[k] // rainfall
			sr += precip[k]
		}

		// collect disaggregated snowmelt distributions (smDist)
		if dt.Hour() == 6 { // start of SNODAS (FEWS-)disaggregated timestep 06:00 UTC
			smk = snowmelt[k] // snowmelt [m/d]
			sm += smk
			smDist = make(map[time.Time]float64, 4) // snow melt distribution
			if smk > 0. {
				smDist = func() map[time.Time]float64 {
					f := make([]float64, 4)
					dtt := make([]time.Time, 4)
					rf := make([]float64, 4)
					tt := make([]float64, 4)
					for i := 0; i < 4; i++ {
						dtt[i] = dt.Add(time.Hour * time.Duration(6*i))
						if temp[k+i] > tc {
							rf[i] = precip[k+i]
						}
						tt[i] = temp[k+i]
					}

					// rule 0: proportion on temperatures > 10°C
					st10 := 0.
					for i := 0; i < 4; i++ {
						if tt[i] > parseRainMeltRule0thresh {
							f[i] = tt[i]
							st10 += tt[i]
						}
					}
					if st10 > 0. {
						for i := 0; i < 4; i++ {
							f[i] /= st10
						}
					} else {
						// rule 1: all to first rainfall > 5mm
						bl := true
						for i := 0; i < 4; i++ {
							if rf[i] > parseRainMeltRule1thresh && bl {
								f[i] = 1.
								bl = false
							} else {
								f[i] = 0. // initialize
							}
						}
						if bl {
							// rule 2: proportion on rainfall > 1mm
							spp := 0.
							for i := 0; i < 4; i++ {
								if rf[i] > parseRainMeltRule2thresh {
									f[i] = rf[i]
									spp += rf[i]
								}
							}
							if spp > 0. {
								for i := 0; i < 4; i++ {
									f[i] /= spp
								}
							} else {
								// rule 3: proportion on temperature > 0
								st0 := 0.
								for i := 0; i < 4; i++ {
									if tt[i] > 0. {
										f[i] = tt[i]
										st0 += tt[i]
									}
								}
								if st0 > 0. {
									for i := 0; i < 4; i++ {
										f[i] /= st0
									}
								} else {
									// rule 4: proportion over daytime
									f[1] = .5
									f[2] = .5
								}
							}
						}
					}

					ff := make(map[time.Time]float64, 4)
					for i := 0; i < 4; i++ {
						ff[dtt[i]] = f[i]
					}
					return ff
				}()

				check := 0.
				for i := 0; i < 4; i++ {
					check += smDist[dt.Add(time.Hour*time.Duration(6*i))]
				}
				if math.Round(check*10000.)/10000. != 1. {
					log.Fatalln("mm: chk: smDist")
				}
			}
		}

		y[k] = yy + smk*smDist[dt] // [m/6hour]
	}

	return y, sp, sm, sr
}
