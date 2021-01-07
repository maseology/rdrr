package main

import (
	"log"
	"time"

	"github.com/maseology/objfunc"
	"github.com/maseology/rdrr/postpro"
)

func evaluate(dts []time.Time, sim []float64, obs postpro.ObsColl) (int, float64, float64, float64) {
	//aggregate
	if len(dts) != len(sim) {
		log.Fatalf("evaluate error: dts and sim must be of same length")
	}

	ndays := int(dte.Sub(dte).Seconds() / 86400.)
	agg := make(map[time.Time]float64, ndays)
	var fobs []float64
	var fdts []time.Time
	for i, t := range obs.T {
		if t.Before(dtb) || t.After(dte) {
			continue
		}
		fobs = append(fobs, obs.V[i])
		fdts = append(fdts, t)
		agg[t] = 0.
	}
	for i, t := range dts {
		d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		if _, ok := agg[t]; ok {
			agg[d] += sim[i] / 4.
		}
	}
	fsim := make([]float64, len(fdts))
	for i, t := range fdts {
		if _, ok := agg[t]; !ok {
			log.Fatalf("error in aggregated date")
		}
		fsim[i] = agg[t]
	}

	kge := objfunc.KGE(fobs, fsim)
	nse := objfunc.NSE(fobs, fsim)
	bias := objfunc.Bias(fobs, fsim)
	return len(fobs), kge, nse, bias
}
