package rdrr

import (
	"time"
)

func daydate(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() // t.Location())
}

func todaily(oo map[int64]float64, ss []float64, ts []time.Time) (obs, sim []float64) {
	obs = make([]float64, len(oo))
	is, c := make(map[int64]int, len(oo)), 0
	for t, v := range oo {
		is[t] = c
		obs[c] = v
		c++
	}

	sim = make([]float64, len(obs))
	sc := make([]int, len(obs))
	for i, t := range ts {
		dd := daydate(t)
		if ii, ok := is[dd]; ok {
			sim[ii] += ss[i]
			sc[ii]++
		}
	}
	for i, c := range sc {
		if c > 1 {
			sim[i] /= float64(c) // mean
		}
	}
	return

	// 	is, c := make(map[time.Time]int, len(oo)), 0
	// 	for t := range oo {
	// 		is[t] = c
	// 		c++
	// 	}
	// 	obs, sim := make([]float64, len(oo)), make([]float64, len(oo))
	// for t,i := range is { // WARNING not in order
	// dd := daydate(t)
	// obs[i] = oo[dd]
	// }

	// 	for _, t := range ts {

	// 		if _, ok := oo[dd]; ok {

	// 		}
	// 	}

	// 	return ss, ss
}
