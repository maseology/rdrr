package prep

import (
	"log"
	"math"
	"time"

	"github.com/maseology/goHydro/pet"
)

// Compute6hourly updates the Cell's state
func (c *Cell) Compute6hourly(tC, pPa, rhf, wvel, ccf, zo float64, dt time.Time) (ep float64) {
	if math.IsNaN(ccf) || math.IsNaN(tC) || math.IsNaN(pPa) || math.IsNaN(rhf) || math.IsNaN(wvel) {
		log.Fatalf("%v NaN found: T=%f  P=%f  wvel=%f  rhf=%f  ccf=%f", dt, tC, pPa, wvel, rhf, ccf)
	}

	// integrate hourly Ep rates for the previous 6-hours
	hr0 := (dt.Hour()-5)%24 - 12 // hours relative to noon
	ep = 0.
	for i := 0; i < 6; i++ {
		Iq := c.SI.PSI(float64(hr0)-.5, dt.YearDay()) // [W/m²] extra-terrestrial radiation for the time of day
		Kg := Iq * (.85 - .47*ccf)                    // [W/m²] pg.151 in DeWalle & Rango; attributed to Linacre (1992) ccf=cloud cover fraction
		// (under the assumption that all surfaces are wet rc=0 and ground energy exchange is negligible G=0)
		h, ea := pet.PenmanMonteith(Kg, tC, pPa, rhf, wvel, 0., zo) // liquid water mass density flux [m/s]
		ep += (h + ea) * 60. * 60.                                  // [m] (total demand over the hour)
	}

	if math.IsNaN(ep) {
		log.Fatalf("%v NaN computed: ep=%f  ", dt, ep)
	}
	return
}
