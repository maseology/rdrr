package model

import "math"

// func (b *subdomain) buildCascadeFraction(p ...float64) map[int]float64 {
// 	// return b.buildCascadeFractionGaussian(p[0])
// 	return b.buildCascadeFractionFuzzy(p[0], p[1], p[2])
// }

// Gaussian variogram model
func (b *subdomain) buildCascadeFractionGaussian(rng float64) map[int]float64 {
	fc := make(map[int]float64, len(b.cids))
	for _, c := range b.cids {
		h := math.Pow(b.strc.TEM.TEC[c].G, 2)
		r := math.Pow(rng, 2)
		fc[c] = (sill-nugget)*(1.-math.Exp(-h/r/a)) + nugget
	}
	return fc
}

// Fuzzy slope
func (b *subdomain) buildCascadeFractionFuzzy(fracMax, mslope float64) map[int]float64 {
	fc := make(map[int]float64, len(b.cids))
	for _, c := range b.cids {
		s := b.strc.TEM.TEC[c].G
		if s <= gradMin {
			fc[c] = 0.
		} else {
			fc[c] = math.Min(fracMax, mslope*math.Log10(s/gradMin))
		}
		// if s <= minslope {
		// 	fc[c] = 0.
		// } else if s >= smax {
		// 	fc[c] = 1.
		// } else {
		// 	fc[c] = math.Log(minslope/s) / math.Log(minslope/smax) // see: fuzzy_slope.xlsx
		// }
	}
	return fc
}
