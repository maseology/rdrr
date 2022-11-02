package model

import "math"

func UcascGaussian(a, s float64) float64 {
	return 1 - math.Exp(s*s/-a)
}

// func UcascLinear() {
// 	if s <= gradMin {
// 		return 0.
// 	} else {
// 		fcasc := mcasc * math.Log10(s/gradMin) //math.Min(fracMax, mcasc*math.Log10(s/gradMin))
// 		// fcasc :=  math.Log(minslope/s) / math.Log(minslope/smax) // see: fuzzy_slope.xlsx
// 		if math.IsInf(fcasc, 0) || fcasc < 0. {
// 			panic("invalid fcasc")
// 		}
// 		if fcasc > maxFcasc {
// 			return maxFcasc
// 		}
// 		return fcasc
// 	}
// }
