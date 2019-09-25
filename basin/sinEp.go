package basin

import "math"

func sinEp(doy int) float64 {
	return (avgEp-minEp)*(1.+math.Sin(2.*math.Pi*float64(doy-offset)/366.-math.Pi/2.)) + minEp // [m]
}
