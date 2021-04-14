package model

import (
	"fmt"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/objfunc"
)

type resulter interface {
	report(bool) []float64
	getTotals(sim, hsto, gsto []float64)
}

type outflow struct {
	sim []float64
}

func (o *outflow) report(dummy bool) []float64     { return o.sim }
func (o *outflow) getTotals(sim, d0, d1 []float64) { o.sim = sim }

type results struct {
	dt                        []time.Time
	obs, sim, hsto, gsto      []float64
	contarea, cellarea, intvl float64
}

func newResults(b *subdomain, nstep int) results {
	var r results
	r.contarea = b.contarea
	r.intvl = b.frc.IntervalSec
	r.cellarea = b.strc.Acell
	return r
}

func (r *results) getTotals(sim, hsto, gsto []float64) {
	r.sim, r.hsto, r.gsto = sim, hsto, gsto
}

func (r *results) report(print bool) []float64 {
	if r.obs == nil {
		return []float64{}
	}

	nse := objfunc.NSE(r.obs, r.sim)
	if print {
		nobs, nsim := func() ([]float64, []float64) {
			ift := int(86400. / r.intvl)
			f := r.cellarea / 86400. // convert to m³/s
			nobs, nsim, ii := make([]float64, len(r.dt)/ift), make([]float64, len(r.dt)/ift), 0
			for k := range r.obs {
				if k%ift == 0 && k > 0 {
					nobs[ii] *= f
					nsim[ii] *= f
					ii++
				}
				nobs[ii] += r.obs[k]
				nsim[ii] += r.sim[k]
			}
			return nobs, nsim
		}()

		kge := objfunc.KGE(nobs[warmup:], nsim[warmup:])
		dnse := objfunc.NSE(nobs[warmup:], nsim[warmup:])
		rmse := objfunc.RMSE(nobs[warmup:], nsim[warmup:])
		mwr2 := objfunc.Krause(computeMonthly(r.dt[warmup:], r.obs[warmup:], r.sim[warmup:], r.intvl, r.contarea))
		bias := objfunc.Bias(r.obs, r.sim)

		fmt.Printf("  KGE: %.3f  NSE: %.3f  RMSE: %.3f  mon-wR²: %.3f  Bias: %.3f\n", kge, dnse, rmse, mwr2, bias)

		mmio.WriteCsvDateFloats("hdgrph.csv", "date,obs,sim", r.dt, r.obs, r.sim)
		mmio.ObsSim("hyd.png", r.obs, r.sim)
		if len(r.obs) < warmup {
			sumPlotSto("sto.png", r.hsto, r.gsto)
		}
	}
	return []float64{1. - nse}
}
