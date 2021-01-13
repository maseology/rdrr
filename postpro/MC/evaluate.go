package main

import (
	"fmt"
	"log"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/objfunc"
	"github.com/maseology/rdrr/postpro"
)

func evaluate(fp string, dts []time.Time, sim []float64, obs postpro.ObsColl) (int, float64, float64, float64) {
	if len(dts) != len(sim) {
		log.Fatalf("evaluate error: dts and sim must be of same length")
	}

	fobs := make([]float64, len(sim))
	c := make(map[time.Time]float64, len(obs.T))
	for i, t := range obs.T {
		c[t] = obs.V[i]
	}
	dd := mmio.DayDate
	for i, t := range dts {
		if v, ok := c[dd(t)]; ok {
			fobs[i] = v
		} else {
			fobs[i] = 0.
		}
	}
	mmio.WriteCsvDateFloats(fmt.Sprintf("%s%s-hdgrph.csv", fp, obs.Nam), "date,obs,sim", dts, fobs, sim)
	fmt.Println(obs.Nam)
	kge := objfunc.KGE(fobs, sim)
	nse := objfunc.NSE(fobs, sim)
	bias := objfunc.Bias(fobs, sim)
	return len(fobs), kge, nse, bias

	// //aggregate
	// ndays := int(dte.Sub(dte).Seconds() / 86400.)
	// agg := make(map[time.Time]float64, ndays)
	// var fobs []float64
	// var fdts []time.Time
	// for i, t := range obs.T {
	// 	if t.Before(dtb) || t.After(dte) {
	// 		continue
	// 	}
	// 	fobs = append(fobs, obs.V[i])
	// 	fdts = append(fdts, t)
	// 	agg[t] = 0.
	// }
	// for i, t := range dts {
	// 	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	// 	if _, ok := agg[t]; ok {
	// 		agg[d] += sim[i] / 4.
	// 	}
	// }
	// fsim := make([]float64, len(fdts))
	// for i, t := range fdts {
	// 	if _, ok := agg[t]; !ok {
	// 		log.Fatalf("error in aggregated date")
	// 	}
	// 	fsim[i] = agg[t]
	// }

	// kge := objfunc.KGE(fobs, fsim)
	// nse := objfunc.NSE(fobs, fsim)
	// bias := objfunc.Bias(fobs, fsim)
	// return len(fobs), kge, nse, bias
}
