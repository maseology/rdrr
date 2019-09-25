package basin

import (
	"fmt"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/objfunc"
)

type resulter interface {
	report() []float64
	getTotals(sim, bf []float64, ytot, atot, rtot, gtot, btot float64)
}

type outflow struct{ sim []float64 }

func (o *outflow) report() []float64                                          { return o.sim }
func (o *outflow) getTotals(sim, dummy []float64, d0, d1, d2, d3, d4 float64) { o.sim = sim }

type results struct {
	dt                             []time.Time
	obs, sim, bf                   []float64
	ytot, atot, rtot, gtot, btot   float64
	h2cms, fncid, fnstrm, contarea float64
	nstep, intvl, ncid, nstrm      int
	// gy, ga, gr, gg, gd, gl        []float64
}

func newResults(b *subdomain, intvl int64, nstep int) results {
	var r results
	r.contarea = b.contarea
	r.h2cms = b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor for subdomain outlet cell
	r.fncid, r.fnstrm = b.fncid, b.fnstrm
	r.nstep, r.intvl, r.ncid, r.nstrm = nstep, int(intvl), b.ncid, b.nstrm
	return r
}

func (r *results) getTotals(sim, bf []float64, ytot, atot, rtot, gtot, btot float64) {
	r.sim, r.bf = sim, bf
	r.ytot, r.atot, r.rtot, r.gtot, r.btot = ytot, atot, rtot, gtot, btot
}
func (r *results) report() []float64 {
	for k := 0; k < r.nstep; k++ {
		r.sim[k] *= r.h2cms / r.fncid
		r.bf[k] *= r.h2cms / r.fnstrm / r.fncid
	}
	rmse := objfunc.RMSE(r.obs[365:], r.sim[365:])
	// sumWriteRMaps(b.mdldir, xr, ds, gy, ga, gr, gg, gd, gl, float64(nstep))
	kge := objfunc.KGE(r.obs, r.sim)
	mwr2 := objfunc.Krause(computeMonthly(r.dt[365:], r.obs[365:], r.sim[365:], float64(r.intvl), r.contarea))
	nse := objfunc.NSE(r.obs, r.sim)
	bias := objfunc.Bias(r.obs, r.sim)
	ff := 365.24 * 1000. / float64(r.nstep) / r.fncid
	fmt.Printf("  waterbudget [mm/yr]: pre: %.0f  aet: %.0f  rch: %.0f  gwd: %.0f  olf: %.0f  dif: %.1f\n", r.ytot*ff, r.atot*ff, r.gtot*ff, r.btot*ff, r.rtot*ff, (r.ytot+r.btot-(r.atot+r.gtot+r.rtot))*ff)
	fmt.Printf("  KGE: %.3f  NSE: %.3f  RMSE: %.3f  mon-wR²: %.3f  Bias: %.3f\n", kge, nse, rmse, mwr2, bias)
	mmio.ObsSim("hyd.png", r.obs, r.sim, r.bf, nil)
	return []float64{rmse}
}
