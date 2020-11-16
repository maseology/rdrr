package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/im7mortal/UTM"
	"github.com/maseology/glbopt"
	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/pet"
	"github.com/maseology/goHydro/solirrad"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
)

const (
	parseRainMeltRule0thresh = 10.
	parseRainMeltRule1thresh = 0.005
	parseRainMeltRule2thresh = 0.001

	penmanWFa, penmanWFb = 0.009, 0.26 // calibrated ep parameters

	gdefFP = "S:/OWRC-RDRR/owrc20-50a.uhdem.gdef"
	demFP  = "S:/OWRC-RDRR/owrc20-50a.uhdem"
	swsFP  = "S:/OWRC-RDRR/owrc20-50a_SWS10.indx"
	ncfp   = "S:/OWRC-RDRR/met/202010010100.nc.bin" // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)

	gobDir = "S:/OWRC-RDRR/met/"
)

var (
	dtb = time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
	dte = time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
)

type cell struct {
	k, cid, swsID int
	psiF          [366]float64
}

func main() {
	tt := mmio.NewTimer()
	defer tt.Print("prep complete!")

	mmio.PrintMemUsage()

	// var dts []time.Time
	// nt, nsta, nvar, dcol := loadNC(ncfp)
	// fmt.Println(nt)
	// fmt.Println(nsta)
	// fmt.Println(nvar)
	// fmt.Println(len(dcol))
	// for s, d := range dcol {
	// 	fmt.Println(s)
	// 	if dts == nil {
	// 		dts = sequentialDates(d)
	// 		fmt.Println(dts[0])
	// 		fmt.Println(dts[len(dts)-1])
	// 	}
	// 	break
	// }

	fmt.Println("\ncollecting DEM..")
	cells, _, nc := getCells(gdefFP, demFP, swsFP)
	mmio.PrintMemUsage()

	fmt.Println("\ncollecting station data and computing basin atmospheric yield and Eao..")
	dts, y, eao, _ := collectStationData(ncfp)
	mmio.PrintMemUsage()

	fmt.Println("converting ys..")
	ys := convertMapToArray(dts, cells, y, nc)
	mmio.PrintMemUsage()

	fmt.Println("adjusting Eao..")
	eas := getAtmosDemandCell(dts, cells, eao, nc)
	mmio.PrintMemUsage()

	fmt.Println("saving met gobs..")
	if err := saveGOB(gobDir+"frc.ys.gob", ys); err != nil {
		log.Fatalf("%v", err)
	}
	if err := saveGOB(gobDir+"frc.ep.gob", eas); err != nil {
		log.Fatalf("%v", err)
	}

	// if 1 == 0 {
	// 	fmt.Println("\ncomputing atmospheric demand..")
	// 	as := getAtmosDemand(dts, cells, c, t, p, r, u, nc, nt)
	// 	fmt.Println("saving demand..")
	// 	if err := saveGOB(gobDir+"frc.ep.gob", as); err != nil {
	// 		log.Fatalf("%v", err)
	// 	}
	// }
}

func getCells(gdefFP, demFP, swsFP string) ([]cell, *grid.Definition, int) {
	gd, err := grid.ReadGDEF(gdefFP, true)
	if err != nil {
		log.Fatalf("%v", err)
	}
	nact := len(gd.Sactives)
	if nact <= 0 {
		log.Fatalf("error: grid definition requires active cells")
	}

	var dem tem.TEM
	if _, ok := mmio.FileExists(demFP + ".TEM.gob"); ok {
		var err error
		fmt.Println(" loading TEM from gob..")
		dem, err = tem.LoadGob(demFP + ".TEM.gob")
		if err != nil {
			log.Fatalf(" tem gob read error: %v", err)
		}
	} else {
		if err := dem.New(demFP); err != nil {
			log.Fatalf(" tem.New() error: %v", err)
		}
		if err := dem.SaveGob(demFP + ".TEM.gob"); err != nil {
			log.Fatalf(" tem.Save() error: %v", err)
		}
	}
	for _, i := range gd.Sactives {
		if dem.TEC[i].Z == -9999. {
			// log.Fatalf("no elevation assigned to cell %d", i)
			fmt.Printf(" WARNING no elevation assigned to meteo cell %d\n", i)
		}
	}

	if _, ok := mmio.FileExists(demFP + ".ContributingCellMap.gob"); !ok {
		fmt.Println(" building contributing cell map gob..")
		ucnt := dem.ContributingCellMap()
		if err := mmio.SaveGOB(demFP+".ContributingCellMap.gob", ucnt); err != nil {
			log.Fatalf(" topo.ContributingCellMap error: %v", err)
		}
	}

	fmt.Println(" +++ MM: SHOULD BE SPAWNING SOME TEM GOBS HERE +++++++++++++++++++")

	fmt.Println(" collecting SWSs..")
	var gsws grid.Indx
	gsws.LoadGDef(gd)
	gsws.New(swsFP, false)
	sws := gsws.Values()

	var cells []cell
	if _, ok := mmio.FileExists(demFP + ".Cells.gob"); ok {
		log.Fatalf("TODO")
	} else {
		fmt.Println(" building cell solar geometry..")
		type in1 struct {
			t      tem.TEC
			k, cid int
			x, y   float64
		}
		generateInput := func(inputStream chan<- in1) {
			for k, cid := range gd.Sactives {
				xy := gd.Coord[cid]
				inputStream <- in1{dem.TEC[cid], k, cid, xy.X, xy.Y}
			}
		}

		newStreamer := func(wg *sync.WaitGroup, done <-chan interface{}, inputStream <-chan in1, outputStream chan<- cell) {
			defer wg.Done()
			go func() {
				for {
					select {
					case s := <-inputStream:
						latitude, _, err := UTM.ToLatLon(s.x, s.y, 17, "", true)
						if err != nil {
							fmt.Println(s)
							log.Fatalf(" newGeomStream error: %v -- (x,y)=(%f, %f); cid: %d\n", err, s.x, s.y, s.cid)
						}
						si := solirrad.New(latitude, math.Tan(s.t.G), math.Pi/2.-s.t.A)
						outputStream <- cell{k: s.k, cid: s.cid, swsID: sws[s.cid], psiF: si.PSIfactor}
					case <-done:
						return
					}
				}
			}()
		}

		done := make(chan interface{})
		inputStream := make(chan in1)
		outputStream := make(chan cell)
		var wg sync.WaitGroup
		wg.Add(64)
		for k := 0; k < 64; k++ {
			newStreamer(&wg, done, inputStream, outputStream)
		}
		go generateInput(inputStream)

		cells = make([]cell, nact)
		for k := 0; k < nact; k++ {
			c := <-outputStream
			cells[c.k] = c
		}
		close(done)
		wg.Wait()
		// close(inputStream)
		// close(outputStream)

		func() error {
			f, err := os.Create(demFP + ".Cells.gob")
			defer f.Close()
			if err != nil {
				log.Fatalf(" cells to gob error: %v", err)
			}
			enc := gob.NewEncoder(f)
			err = enc.Encode(cells)
			if err != nil {
				return err
			}
			return nil
		}()
	}

	return cells, gd, nact
}

func collectStationData(ncfp string) (dts []time.Time, y, ea map[int][]float64, ndat int) {
	fmt.Println(" loading data exported from FEWS:", ncfp, "...")

	ndat, nsta, nvar, dat := loadNC(ncfp) //[station_id][datetime][ precipitation_amount, surface_snow_and_ice_melt_flux, air_temperature, air_pressure, relative_humidity, wind_speed ]
	fmt.Printf("  %6d stations\n %6d timesteps\n %6d variables\n\nParsing precipitation form..\n", nsta, ndat, nvar)

	// yield, temperature and pressure is acquired on a sws basis
	y = make(map[int][]float64, nsta)
	ea = make(map[int][]float64, nsta)
	// c = make(map[int]map[time.Time]float64, nsta)
	// t = make(map[int]map[time.Time]float64, nsta)
	// p = make(map[int]map[time.Time]float64, nsta)
	// r = make(map[int]map[time.Time]float64, nsta)
	// u = make(map[int]map[time.Time]float64, nsta)
	// v = make(map[int]map[time.Time]float64, nsta)

	for sid, d := range dat {
		if dts == nil {
			dts = sequentialDates(d)
		}
		prec, smlt, temp, eao := make([]float64, len(dts)), make([]float64, len(dts)), make([]float64, len(dts)), make([]float64, len(dts))
		tl, pl, rhl := 10., 101.3, .85

		for i, dt := range dts {
			if _, ok := d[dt]; !ok {
				log.Fatalf("Date %v not found in %s\n", dt, ncfp)
			}
			vs := d[dt]
			prec[i] = float64(vs[0])
			smlt[i] = float64(vs[1])
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

		y[sid] = getAtomsYieldByStation(sid, dts, prec, smlt, temp) // expensive
		ea[sid] = eao

		// c[sid] = prec
		// t[sid] = temp
		// p[sid] = pres
		// r[sid] = rhfc
		// u[sid] = wvel
	}
	return
}

func loadNC(ncfp string) (int, int, int, map[int]map[time.Time][]float32) {
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

		dcol := make(map[int]map[time.Time][]float32, nsta)
		for i := 0; i < nsta; i++ {
			d := make(map[time.Time][]float32, nt)
			sid := int(mmio.ReadInt32(b))
			// fmt.Println(sid)
			for j := 0; j < nt; j++ {
				a := make([]float32, nvar)
				for iv := 0; iv < nvar; iv++ {
					a[iv] = mmio.ReadFloat32(b)
				}
				// copy(d[times[j]], a)
				d[times[j]] = a
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

	// spp := sm + sr
	// dd := float64(len(dts)) / 365.24 / 4.
	// fmt.Printf("%10d: %10.1f %10.1f (m%.1f r%.1f) %20.5f %20.5f\n", sid, sp/dd, spp/dd, sm/dd, sr/dd, (spp-sp)/sp, tc) // mm/yr

	return
}

func parseRainMelt(orderedDatetimeList []time.Time, precip, snowmelt, temp []float64, tc float64) ([]float64, float64, float64, float64) {
	y := make([]float64, len(precip))
	sp, sm, sr := 0., 0., 0.
	for k, dt := range orderedDatetimeList {
		// fmt.Println(dt)
		yy := 0.0
		sp += precip[k]
		var smDist map[time.Time]float64
		if temp[k] >= tc {
			yy += precip[k] // rainfall
			sr += precip[k]
		}

		// collect disaggregated snowmelt distributions (smDist)
		if dt.Hour() == 6 { // start of SNODAS (FEWS-)disaggregated timestep 06:00 UTC
			sm += snowmelt[k]                       // snowmelt (m/d)
			smDist = make(map[time.Time]float64, 4) // snow melt distribution
			if snowmelt[k] > 0. {
				smDist = func() map[time.Time]float64 {
					f := make([]float64, 4)
					dtt := make([]time.Time, 4)
					rf := make([]float64, 4)
					tt := make([]float64, 4)
					for i := 0; i < 4; i++ {
						dtt[i] = dt.Add(time.Hour * time.Duration(6*i))
						// if temp[dtt[i]] > tc {
						// 	rf[i] = precip[dtt[i]]
						// 	tt[i] = temp[dtt[i]]
						// }
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

		if smDist != nil {
			yy += snowmelt[k] * smDist[dt] * 1000. // snowmelt (m/d)
		}

		y[k] = yy
	}
	if sp == 0. {
		print()
	}
	return y, sp, sm, sr
}

func convertMapToArray(dts []time.Time, cs []cell, y map[int][]float64, nc int) [][]float64 {
	ys := make([][]float64, nc)
	nt := len(dts)
	for i := 0; i < nc; i++ {
		ys[i] = make([]float64, nt)
		if _, ok := y[cs[i].swsID]; !ok {
			log.Fatalf("Data check convertMapToArray, invalid sws id: %d", cs[i].swsID)
		}
	}
	fmt.Println("blh")
	for k, dt := range dts {
		if dt.After(dte) || dt.Before(dtb) {
			fmt.Printf("blh")
			continue
		}
		for i := 0; i < nc; i++ {
			c := cs[i]
			ys[i][k] = y[c.swsID][k]
		}
	}
	return ys
}

func getAtmosDemandCell(dts []time.Time, cs []cell, eao map[int][]float64, nc int) [][]float64 {
	as := make([][]float64, nc) // [cell ID][date ID]
	nt := len(dts)
	for i := 0; i < nc; i++ {
		as[i] = make([]float64, nt)
	}

	for k, dt := range dts {
		if dt.After(dte) || dt.Before(dtb) {
			continue
		}
		// fmt.Println(dt.Format("2006-01-02 15:04:05"))
		// dtD := time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, dt.Location())
		for i := 0; i < nc; i++ {
			c := cs[i]
			as[i][k] = c.psiF[dt.YearDay()-1] * eao[c.swsID][k] // [m/6hr]
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
