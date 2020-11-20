package model

import (
	"math"
	"time"

	"github.com/maseology/mmio"
)

func computeMonthly(dt []time.Time, o, s []float64, ts, ca float64) ([]float64, []float64) {
	tso, tss := make(mmio.TimeSeries, len(dt)), make(mmio.TimeSeries, len(dt))
	for i, d := range dt {
		if math.IsNaN(o[i]) || math.IsNaN(s[i]) {
			continue
		}
		tso[d] = o[i]
		tss[d] = s[i]
	}
	os, _ := mmio.MonthlySumCount(tso)
	ss, _ := mmio.MonthlySumCount(tss)
	dn, dx := mmio.MinMaxTimeseries(tso)
	i := 0
	osi, ssi := make([]float64, len(os)*12), make([]float64, len(ss)*12)
	cf := ts * 1000. / ca // sum(cms) to mm/mo
	for y := mmio.Yr(dn.Year()); y <= mmio.Yr(dx.Year()); y++ {
		for m := mmio.Mo(1); m <= 12; m++ {
			if v, ok := os[y][m]; ok {
				if math.IsNaN(v) || math.IsNaN(ss[y][m]) {
					continue
				}
				osi[i] = v * cf
				ssi[i] = ss[y][m] * cf
				i++
			}
		}
	}
	return osi, ssi
}
