package basin

import "math"

const (
	avgEp  = 1. / 366. // average annual potential evaporation [m/day]
	minEp  = 0.        // baseline evaporation rate [m/day]
	offset = 10        // offset to date of min Ep (adjusts the winter solstice 10 days before new years, i.e., Dec-21 'see sinET.xlsx)
)

func sinEp(doy int) float64 {
	return (avgEp-minEp)*(1.+math.Sin(2.*math.Pi*float64(doy-offset)/366.-math.Pi/2.)) + minEp // [m]
}
