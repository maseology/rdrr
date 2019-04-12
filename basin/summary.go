package basin

import (
	"math"
	"time"

	"github.com/maseology/mmio"
)

func sumHydrograph(dt, o, s, g, x []interface{}) {
	// C:/Users/mason/OneDrive/R/dygraph/obssim_csv_viewer.R
	mmio.WriteCSV("hydrograph.csv", "date,obs,sim,gw,excess", dt, o, s, g, x)
	// mmio.ObsSim("hydrograph.png", o[730:], s[730:])
}

func sumMonthly(dt, o, s []interface{}, ts, ca float64) {
	tso, tss := make(mmio.TimeSeries, len(dt)), make(mmio.TimeSeries, len(dt))
	for i, d := range dt {
		if o[i] == nil || s[i] == nil {
			continue
		}
		dt1 := d.(time.Time)
		tso[dt1] = o[i].(float64)
		tss[dt1] = s[i].(float64)
	}
	os, _ := mmio.MonthlySumCount(tso)
	ss, _ := mmio.MonthlySumCount(tss)
	dn, dx := mmio.MinMaxTimeseries(tso)
	dti, i := make([]interface{}, len(os)*12), 0
	osi, ssi := make([]interface{}, len(os)*12), make([]interface{}, len(ss)*12)
	for y := mmio.Yr(dn.Year()); y <= mmio.Yr(dx.Year()); y++ {
		for m := mmio.Mo(1); m <= 12; m++ {
			if v, ok := os[y][m]; ok {
				if math.IsNaN(v) || math.IsNaN(ss[y][m]) {
					continue
				}
				dti[i] = time.Date(int(y), m, 15, 0, 0, 0, 0, time.UTC)
				cf := ts * 1000. / ca // sum(cms) to mm/mo
				osi[i] = v * cf
				ssi[i] = ss[y][m] * cf
				i++
			}
		}
	}
	mmio.WriteCSV("monthlysum.csv", "date,obs,sim", dti, osi, ssi)
}
