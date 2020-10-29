package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"time"

	"github.com/im7mortal/UTM"
	"github.com/maseology/glbopt"
	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/prep"
)

const (
	rule0thresh       = 10.
	rule1thresh       = 0.005
	rule2thresh       = 0.001
	cloudcoverthresh  = 0.001
	b, g, alpha, beta = .06142, .899, 1.3077261, -0.000361 // calibrated PE parameters

	gdefFP = "M:/OWRC-RDRR/owrc20-50a.uhdem.gdef"
	demFP  = "M:/OWRC-RDRR/owrc20-50a.uhdem"
	swsFP  = "M:/OWRC-RDRR/owrc20-50a_SWS10.indx"
	ncfp   = "M:/OWRC-RDRR/met/202010010100.nc.bin" // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)

	gobDir = "M:/OWRC-RDRR/met/"
)

var (
	dtb = time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
	dte = time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
)

func main() {
	tt := mmio.NewTimer()
	defer tt.Print("prep complete!")

	fmt.Println("\ncomputing atmospheric yield..")
	dts, cells, y, c, t, p, r, u, nc, nt := collectStationData()

	fmt.Println("\ncomputing atmospheric demand..")
	as := getAtmosDemand(dts, cells, c, t, p, r, u, nc, nt)
	ys := getAtomsYieldByCell(dts, cells, y, nc, nt)

	// build .gob
	fmt.Println("\nsaving..")
	if err := saveGOB(gobDir+"frc.y.gob", ys); err != nil {
		log.Fatalf("%v", err)
	}
	if err := saveGOB(gobDir+"frc.ep.gob", as); err != nil {
		log.Fatalf("%v", err)
	}
}

func loadNC(ncfp string) (int, int, int, map[int]map[time.Time][]float32) {
	switch ext := mmio.GetExtension(ncfp); ext {
	case ".bin":
		b := mmio.OpenBinary(ncfp)
		nt := int(mmio.ReadInt32(b))
		times := make([]time.Time, nt)
		for i := 0; i < nt; i++ {
			times[i] = time.Unix(mmio.ReadInt64(b), 0).UTC()
		}

		nsta := int(mmio.ReadInt32(b))
		nvar := int(mmio.ReadInt32(b))

		dcol := make(map[int]map[time.Time][]float32, nsta)
		for i := 0; i < nsta; i++ {
			d := make(map[time.Time][]float32, nt)
			sid := int(mmio.ReadInt32(b))
			// fmt.Println(sid)
			for j := 0; j < nt; j++ {
				d[times[j]] = []float32{mmio.ReadFloat32(b), mmio.ReadFloat32(b), mmio.ReadFloat32(b), mmio.ReadFloat32(b), mmio.ReadFloat32(b), mmio.ReadFloat32(b)}
			}
			dcol[sid] = d
			// break
		}

		return nt, nsta, nvar, dcol
	case ".nc":
		log.Fatalln("ERROR: current not supporting NetCDF files. Convert to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py.")
	default:
		log.Fatalln("ERROR: loadNC unsupported file type:" + ext)
	}
	return -1, -1, -1, nil
}

func sequentialDates(d map[time.Time][]float32) []time.Time {
	dts := make([]time.Time, len(d))
	i := 0
	for k := range d {
		dts[i] = k
		i++
	}
	sort.Slice(dts, func(i, j int) bool { return dts[i].Before(dts[j]) })
	return dts
}

func cleanData(orderedDatetimeList []time.Time, precip, snowmelt, temp, pres, rhfc, wvel map[time.Time]float64) {
	tlast, plast, rhlast := 0., 101.3, .5
	for _, dt := range orderedDatetimeList {
		if _, ok := precip[dt]; !ok {
			precip[dt] = 0.
		}
		if math.IsNaN(precip[dt]) {
			precip[dt] = 0.
		}
		if _, ok := snowmelt[dt]; !ok {
			snowmelt[dt] = 0.
		}
		if math.IsNaN(snowmelt[dt]) {
			snowmelt[dt] = 0.
		}
		if _, ok := temp[dt]; !ok {
			temp[dt] = tlast
		}
		if math.IsNaN(temp[dt]) {
			temp[dt] = tlast
		}
		tlast = temp[dt]
		if _, ok := pres[dt]; !ok {
			pres[dt] = plast
		}
		if math.IsNaN(pres[dt]) {
			pres[dt] = plast
		}
		plast = pres[dt]
		if _, ok := rhfc[dt]; !ok {
			rhfc[dt] = rhlast
		}
		if math.IsNaN(rhfc[dt]) {
			rhfc[dt] = rhlast
		}
		rhlast = rhfc[dt]
		if rhfc[dt] < 0. || rhfc[dt] > 1. {
			log.Fatalln("RH is to be entered as a fraction [0,1]")
		}
		if _, ok := wvel[dt]; !ok {
			wvel[dt] = 0.
		}
		if math.IsNaN(wvel[dt]) {
			wvel[dt] = 0.
		}
	}
}

func collectStationData() (dts []time.Time, cells []prep.Cell, y, c, t, p, r, u map[int]map[time.Time]float64, nc, nt int) {
	fmt.Println("\nloading data exported from FEWS:", ncfp, "...")
	nt, nsta, nvar, dat := loadNC(ncfp) //[station_id][datetime][ precipitation_amount, surface_snow_and_ice_melt_flux, air_temperature, air_pressure, relative_humidity, wind_speed ]
	fmt.Printf(" %6d stations\n %6d timesteps\n %6d variables\n\n", nsta, nt, nvar)

	// yield, temperature and pressure is acquired on a sws basis
	y = make(map[int]map[time.Time]float64, nsta)
	c = make(map[int]map[time.Time]float64, nsta)
	t = make(map[int]map[time.Time]float64, nsta)
	p = make(map[int]map[time.Time]float64, nsta)
	r = make(map[int]map[time.Time]float64, nsta)
	u = make(map[int]map[time.Time]float64, nsta)
	for sid, d := range dat {
		prec, smlt, temp, pres, rhfc, wvel := make(map[time.Time]float64, len(d)), make(map[time.Time]float64, len(d)), make(map[time.Time]float64, len(d)), make(map[time.Time]float64, len(d)), make(map[time.Time]float64, len(d)), make(map[time.Time]float64, len(d))
		if dts == nil {
			dts = sequentialDates(d)
		}
		for dt, vs := range d {
			prec[dt] = float64(vs[0])
			smlt[dt] = float64(vs[1])
			temp[dt] = float64(vs[2])
			pres[dt] = float64(vs[3])
			rhfc[dt] = float64(vs[4])
			wvel[dt] = float64(vs[5])
		}
		cleanData(dts, prec, smlt, temp, pres, rhfc, wvel)

		y[sid] = getAtomsYield(sid, dts, prec, smlt, temp)
		c[sid] = prec
		t[sid] = temp
		p[sid] = pres
		r[sid] = rhfc
		u[sid] = wvel
	}

	// demand on a cell basis
	cells, _, nc = getCells()
	return
}

func parseRainMelt(orderedDatetimeList []time.Time, precip, snowmelt, temp map[time.Time]float64, tc float64) (map[time.Time]float64, float64, float64, float64) {
	y := make(map[time.Time]float64)
	sp, sm, sr := 0., 0., 0.
	for _, dt := range orderedDatetimeList {
		// fmt.Println(dt)
		yy := 0.0
		sp += precip[dt]
		var smDist map[time.Time]float64
		if temp[dt] >= tc {
			yy += precip[dt] // rainfall
			sr += precip[dt]
		}

		// collect disaggregated snowmelt distributions (smDist)
		if dt.Hour() == 6 { // start of SNODAS (FEWS-)disaggregated timestep 06:00 UTC
			sm += snowmelt[dt] // snowmelt (m/d)
			smDist = make(map[time.Time]float64)
			if snowmelt[dt] <= 0. {
				for i := 0; i < 4; i++ {
					smDist[dt.Add(time.Hour*time.Duration(6*i))] = 0.
				}
			} else {
				// if dt.Year() == 2020 && dt.Month() == 8 && dt.Day() == 21 {
				// 	fmt.Println(dt)
				// }
				smDist = func() map[time.Time]float64 {
					f := make([]float64, 4)
					dtt := make([]time.Time, 4)
					rf := make([]float64, 4)
					tt := make([]float64, 4)
					for i := 0; i < 4; i++ {
						dtt[i] = dt.Add(time.Hour * time.Duration(6*i))
						if temp[dtt[i]] > tc {
							rf[i] = precip[dtt[i]]
							tt[i] = temp[dtt[i]]
						}
					}

					// rule 0: proportion on temperatures > 10°C
					st10 := 0.
					for i := 0; i < 4; i++ {
						if tt[i] > rule0thresh {
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
							if rf[i] > rule1thresh && bl {
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
								if rf[i] > rule2thresh {
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

		if smDist != nil {
			yy += snowmelt[dt] * smDist[dt] * 1000. // snowmelt (m/d)
		}

		y[dt] = yy
	}

	return y, sp, sm, sr
}

func getAtomsYield(sid int, dts []time.Time, p, m, t map[time.Time]float64) (y map[time.Time]float64) {
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

	dd := float64(len(dts)) / 365.24 / 4.
	fmt.Printf("%10d: %10.1f %10.1f (m%.1f r%.1f) %20.5f %20.5f\n", sid, sp/dd, spp/dd, sm/dd, sr/dd, (spp-sp)/sp, tc) // mm/yr

	return
}

func getAtomsYieldByCell(dts []time.Time, cells []prep.Cell, y map[int]map[time.Time]float64, nc, nt int) [][]float64 {
	ys := make([][]float64, nc)
	for i := 0; i < nc; i++ {
		ys[i] = make([]float64, nt)
	}
	for k, dt := range dts {
		if dt.After(dte) || dt.Before(dtb) {
			continue
		}
		for i := 0; i < nc; i++ {
			c := cells[i]
			ys[i][k] = y[c.SwsID][dt]
		}
	}
	return ys
}

func getCells() ([]prep.Cell, *grid.Definition, int) {
	gd, err := grid.ReadGDEF(gdefFP, true)
	if err != nil {
		log.Fatalf("%v", err)
	}
	nact := len(gd.Sactives)
	if nact <= 0 {
		log.Fatalf("error: grid definition requires active cells")
	}

	tem, err := tem.NewTEM(demFP)
	if err != nil {
		log.Fatalf("%v", err)
	}
	for _, i := range gd.Sactives {
		if tem.TEC[i].Z == -9999. {
			// log.Fatalf("no elevation assigned to cell %d", i)
			fmt.Printf(" WARNING no elevation assigned to meteo cell %d\n", i)
		}
	}
	fmt.Println("MM: SHOULD BE SPAWNING SOME TEM GOBS HERE +++++++++++++++++++")

	var gsws grid.Indx
	gsws.LoadGDef(gd)
	gsws.New(swsFP, false)
	sws := gsws.Values()

	cells := make([]prep.Cell, nact)
	for k, i := range gd.Sactives {
		latitude, _, err := UTM.ToLatLon(gd.Coord[i].X, gd.Coord[i].Y, 17, "", true)
		if err != nil {
			log.Fatalf(" prep error: %v -- (x,y)=(%f, %f); cid: %d\n", err, gd.Coord[i].X, gd.Coord[i].Y, i)
		}
		t := tem.TEC[i]
		cells[k] = prep.NewCell2(latitude, math.Tan(t.G), math.Pi/2.-t.A, b, g, alpha, beta)
		cells[k].SwsID = sws[i]
	}
	return cells, gd, nact
}

func getAtmosDemand(dts []time.Time, cells []prep.Cell, precip, temperature, pressure, rh, wvel map[int]map[time.Time]float64, nc, nt int) [][]float64 {
	as := make([][]float64, nc) // [cell ID][date ID]
	for i := 0; i < nc; i++ {
		as[i] = make([]float64, nt)
	}

	for k, dt := range dts {
		if dt.After(dte) || dt.Before(dtb) {
			continue
		}
		fmt.Println(dt)
		for i := 0; i < nc; i++ {
			c, cf := cells[i], 0.
			pp := precip[c.SwsID][dt]
			if pp > cloudcoverthresh {
				cf = 1.
			} else if pp > 0. {
				cf = pp / cloudcoverthresh
			}
			as[i][k] = c.Compute6hourly(temperature[c.SwsID][dt], pressure[c.SwsID][dt], rh[c.SwsID][dt], wvel[c.SwsID][dt], cf, dt)
		}
	}
	return as
}

func saveGOB(fp string, d [][]float64) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(d)
	if err != nil {
		return err
	}
	return nil
}
