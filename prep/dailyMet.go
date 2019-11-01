package prep

import (
	"log"
	"math"
	"time"

	"github.com/maseology/goHydro/pet"
)

const (
	// a = .27
	// b = .52
	// // Prescott
	// a     = .37503
	// b     = .68627
	// t     = .0007986
	// nNn   = .2732
	// alpha = .6783
	// beta  = -0.00097315

	// Bristow Campbell
	a     = 1.
	b     = .06142
	g     = .899
	alpha = 1.3077261
	beta  = -0.000361
)

// func etRadToGlobal(Ke, nN float64) float64 {
// 	// the Prescott-type equation (Novák, 2012, pg.232)
// 	return Ke * (a + b*nN)
// }

func etRadToGlobal(Ke, tx, tn float64) float64 {
	// see pg 151 in DeWalle & Rango; attributed to Bristow and Campbell (1984)
	// ref: Bristow, K.L. and G.S. Campbell, 1984. On the relationship between incoming solar radiation and daily maximum and minimum temperature. Agricultural and Forest Meteorology 31(2):159--166.
	Kg := Ke * a * (1. - math.Exp(-b*math.Pow(tx-tn, g)))
	return Kg
}

// ComputeDaily updates the Cell's state
func (c *Cell) ComputeDaily(rain, snow, tn, tx float64, dt time.Time) (y, ep float64) {
	if math.IsNaN(rain) || math.IsNaN(snow) || math.IsNaN(tn) || math.IsNaN(tx) {
		log.Fatalf("%v NaN found: Rf=%f  Sf=%f  Tn=%f  Tx=%f  ", dt, rain, snow, tn, tx)
	}
	tm, doy := (tx+tn)/2., dt.YearDay()
	if math.IsNaN(tm) {
		log.Fatalf("%v NaN found: tx=%f  tn=%f  ", dt, tx, tn)
	}
	if tn > tx {
		tn = tx - 0.01
	}
	y = c.SP.Update(rain, snow, tm)
	// nN := 1. // ratio of sunshine hours (n) to total possible ( N = si.DaylightHours(doy) )
	// if rain+snow > t {
	// 	nN = 0.
	// }
	Kg := etRadToGlobal(c.SI.PSIdaily(doy), tx, tn)
	ep = pet.Makkink(Kg, tm, 101300., alpha, beta)
	if math.IsNaN(y) || math.IsNaN(ep) {
		log.Fatalf("%v NaN computed: yeild=%f  ep=%f  ", dt, y, ep)
	}
	return
}
