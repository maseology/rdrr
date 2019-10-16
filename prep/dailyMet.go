package prep

import (
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

func (c *Cell) ComputeDaily(rain, snow, tn, tx float64, doy int) (y, ep float64) {
	tm := (tx + tn) / 2.
	y = c.SP.Update(rain, snow, tm)
	nN := 1. // ratio of sunshine hours (n) to total possible ( N = si.DaylightHours(doy) )
	if rain+snow > 0. {
		nN = 0.
	}
	Kg := etRadToGlobal(c.SI.PSIdaily(doy), nN)
	ep = pet.Makkink(Kg, tm, 101300.)
	return
}
