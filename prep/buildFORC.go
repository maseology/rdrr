package prep

import (
	"fmt"
	"log"
	"math"
	"time"

	"rdrr2/model"

	"github.com/maseology/goHydro/gmet"
	"github.com/maseology/mmio"
)

const intvl = 86400 / 4

// BuildFORC builds the gob containing forcing data: 1) loads FEWS NetCDF (bin) output; 2) returns sorted dates; 3) computes basin; 4) parses precipitation into rainfall by optimizing t_crit
func BuildFORC(gobDir, ncfp string, cmxr map[int]int, outlets []int, carea float64) *model.FORC {
	tt := mmio.NewTimer()

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
	rf := g.GetAllData("rainfall_amount")
	sm := g.GetAllData("surface_snow_melt_amount")
	eao := g.GetAllData("water_potential_evaporation_amount")

	// collect subset of met IDs
	ssmid := make(map[int]bool)
	for _, mid := range cmxr {
		if _, ok := ssmid[mid]; !ok {
			ssmid[mid] = true
		}
	}
	nsta := len(ssmid)

	min0 := func(x float64, s string, m int, t time.Time) float64 {
		if math.IsNaN(x) {
			fmt.Printf(" %s Nan (%d): %v -- set to zero\n", s, m, t)
			return 0.
		}
		if x < 0. {
			return 0.
		}
		return x / 1000. // to [m]
	}
	ys, es, ii, el := make([][]float64, nsta), make([][]float64, nsta), 0, 0.
	mixr := make(map[int]int)
	for i, mid := range g.Sids {
		if _, ok := ssmid[mid]; !ok {
			continue
		}
		ya, ea := make([]float64, g.Nts), make([]float64, g.Nts)
		for j, t := range g.Ts {
			ya[j] = min0(rf[i][j], "rf", mid, t) + min0(sm[i][j], "sf", mid, t)
			if eao[i][j] >= 0. {
				el = eao[i][j]
			} else {
				eao[i][j] = el
			}
			ea[j] = eao[i][j] / 1000. // to [m]
		}
		ys[ii] = ya
		es[ii] = ea
		mixr[mid] = ii
		ii++
	}
	cixr := make(map[int]int, len(cmxr))
	for c, m := range cmxr {
		cixr[c] = mixr[m]
	}

	frc := model.FORC{
		T:           g.Ts,
		Ya:          ys,
		Ea:          es,
		XR:          cixr,
		IntervalSec: intvl,
	}

	if err := frc.SaveGob(gobDir + "domain.FORC.gob"); err != nil {
		log.Fatalf(" BuildFORC error: %v", err)
	}

	tt.Lap("FORC loaded")
	return &frc
}
