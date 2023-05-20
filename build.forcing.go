package rdrr

import (
	"fmt"
	"math"
	"time"

	"github.com/maseology/goHydro/gmet"
)

func buildForcings(mids []int, ncfp string) Forcing {
	tt := time.Now()

	vars := []string{
		// "air_temperature",
		// "air_pressure",
		// "relative_humidity",
		// "wind_speed",
		"water_potential_evaporation_amount",
		"rainfall_amount",
		// "snowfall_amount",
		"surface_snow_melt_amount",
	}
	fmt.Println("loading: " + ncfp)
	g, _ := gmet.LoadNC(ncfp, vars)
	fmt.Printf("  Dates available: %v to %v\n", g.Ts[0], g.Ts[g.Nts-1])
	// collect sequential dates
	ts, xt, nts := func() ([]time.Time, []int, int) {
		d := make(map[int64]int, g.Nts)
		for j, t := range g.Ts {
			d[t.Unix()] = j
		}

		dt, cdt := g.Ts[0], 0
		for {
			if _, ok := d[dt.Unix()]; !ok {
				fmt.Printf("   > missing date %v\n", dt)
				d[dt.Unix()] = -1
				cdt++
			}
			dt = dt.Add(time.Second * time.Duration(intvl))
			if dt.After(g.Ts[g.Nts-1]) {
				break
			}
		}
		if cdt > 0 {
			fmt.Printf("     Total missing dates = %d\n", cdt)
		}

		o, x := make([]time.Time, 0, len(d)), make([]int, 0, len(d))
		dt = g.Ts[0]
		for {
			if xx, ok := d[dt.Unix()]; ok {
				x = append(x, xx)
			} else {
				panic("FORC sequential dates error")
			}
			o = append(o, dt)
			dt = dt.Add(time.Second * time.Duration(intvl))
			if dt.After(g.Ts[g.Nts-1]) {
				break
			}
		}
		fmt.Printf("  Dates available2: %v to %v\n", o[0], o[len(o)-1])
		return o, x, len(o)
	}()

	// collect data
	rf := g.GetAllData("rainfall_amount")
	sm := g.GetAllData("surface_snow_melt_amount")
	eao := g.GetAllData("water_potential_evaporation_amount")

	// collect subset of met IDs
	mmid := make(map[int]int, len(g.Sids))
	for i, s := range g.Sids {
		mmid[s] = i
	}

	ys, es, el := make([][]float64, len(mids)), make([][]float64, len(mids)), .001
	min0 := func(x float64, s string, m int, t time.Time) float64 {
		if math.IsNaN(x) {
			fmt.Printf("   > %s Nan (%d): %v -- set to zero\n", s, m, t)
			return 0.
		}
		if x < 0. {
			return 0.
		}
		return x / 1000. // to [m]
	}
	for ii, s := range mids {
		if i, ok := mmid[s]; ok {
			ya, ea := make([]float64, nts), make([]float64, nts)
			for j, t := range ts {
				jj := xt[j]
				if jj >= 0 {
					ya[j] = min0(rf[i][jj], "rf", s, t) + min0(sm[i][jj], "sf", s, t)
				}
				if jj >= 0 && eao[i][jj] >= 0. {
					ea[j] = eao[i][jj] / 1000. // to [m]
					el = ea[j]
				} else {
					ea[j] = el // infilling with last
				}
			}
			ys[ii] = ya
			es[ii] = ea
		} else {
			panic("loadForcings error: met index does not contain sws index")
		}
	}

	frc := Forcing{
		T:  ts,
		Ya: ys,
		Ea: es,
		// XR:          cixr,
		IntervalSec: intvl,
	}

	fmt.Printf(" Forcing loaded - %v\n", time.Now().Sub(tt))
	return frc
}
