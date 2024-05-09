package forcing

import (
	"fmt"
	"time"

	"github.com/maseology/goHydro/gmet"
	"github.com/maseology/mmio"
)

func GetForcings(mids []int, intvl float64, offset int, ncfp, prfx string) Forcing {
	tt := time.Now()

	fmt.Println(" loading: " + ncfp)
	g := func(fp string) *gmet.GMET {
		var g *gmet.GMET
		var err error
		switch mmio.GetExtension(fp) {
		case ".nc":
			// vars := []string{"precipitation_amount"}
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
			g, err = gmet.LoadNC(fp, prfx, vars)
		case ".csv":
			g, err = gmet.LoadCsv(fp, "precipitation_amount")
		default:
			panic("unknown frc type")
		}
		if err != nil {
			panic(err)
		}
		return g
	}(ncfp)

	frc := build(g, mids, intvl, offset)

	fmt.Printf(" Forcing loaded - %v\n", time.Since(tt))
	return frc
}
