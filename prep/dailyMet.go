package prep

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/maseology/goHydro/pet"
)

const (
	a = .27
	b = .52
)

func etRadToGlobal(Ke, nN float64) float64 {
	// the Prescott-type equation (Novák, 2012, pg.232)
	return Ke * (a + b*nN)
}

// ComputeDaily updates the Cell's state
func (c *Cell) ComputeDaily(rain, snow, tn, tx float64, dt time.Time) (y, ep float64) {
	if math.IsNaN(rain) || math.IsNaN(snow) || math.IsNaN(tn) || math.IsNaN(tx) {
		log.Fatalf("%v NaN found: Rf=%f  Sf=%f  Tn=%f  Tx=%f  ", dt, rain, snow, tn, tx)
	}
	tm, doy := (tx+tn)/2., dt.YearDay()
	if math.IsNaN(tm) {
		fmt.Println("blah")
	}
	y = c.SP.Update(rain, snow, tm)
	nN := 1. // ratio of sunshine hours (n) to total possible ( N = si.DaylightHours(doy) )
	if rain+snow > .001 {
		nN = 0.
	}
	Kg := etRadToGlobal(c.SI.PSIdaily(doy), nN)
	ep = pet.Makkink(Kg, tm, 101300.)
	if math.IsNaN(y) || math.IsNaN(ep) {
		log.Fatalf("%v NaN computed: yeild=%f  ep=%f  ", dt, y, ep)
	}
	return
}
