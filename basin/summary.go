package basin

import (
	"math"
	"time"

	"github.com/maseology/mmio"
)

func computeMonthly(dt []interface{}, o, s []float64, ts, ca float64) ([]float64, []float64) {
	tso, tss := make(mmio.TimeSeries, len(dt)), make(mmio.TimeSeries, len(dt))
	for i, d := range dt {
		if math.IsNaN(o[i]) || math.IsNaN(s[i]) {
			continue
		}
		dt1 := d.(time.Time)
		tso[dt1] = o[i]
		tss[dt1] = s[i]
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

func sumHydrograph(dt, o, s, b []interface{}) {
	// C:/Users/mason/OneDrive/R/dygraph/obssim_csv_viewer.R
	mmio.WriteCSV("hydrograph.csv", "date,obs,sim,bf", dt, o, s, b)
	// mmio.ObsSim("hydrograph.png", o[730:], s[730:])
	// xs, ys := make([]float64, len(s)), make(map[string][]float64, 3)
	// for i := range s {
	// 	xs[i] = float64(i)
	// }
	// ys["obs"] = mmio.InterfaceToFloat(o)
	// ys["sim"] = mmio.InterfaceToFloat(s)
	// ys["bf"] = mmio.InterfaceToFloat(b)
	// mmio.Line("hydrograph.png", xs, ys)
}

func sumHydrographWB(dt, s, d, a, g, x, k []interface{}) {
	// C:/Users/mason/OneDrive/R/dygraph/obssim_csv_viewer.R
	mmio.WriteCSV("waterbalance.csv", "date,sto,dfc,aet,rch,exs,lag", dt, s, d, a, g, x, k)
}

func sumPlotHydrograph(fp string, o, s, b, x []interface{}) {
	xs, ys := make([]float64, len(s)), make(map[string][]float64, 4)
	for i := range s {
		xs[i] = float64(i)
	}
	ys["obs"] = mmio.InterfaceToFloat(o)
	ys["sim"] = mmio.InterfaceToFloat(s)
	ys["bf"] = mmio.InterfaceToFloat(b)
	ys["xs"] = mmio.InterfaceToFloat(x)
	mmio.Line(fp, xs, ys)
}

func sumPlotHydrographWB(fp string, s, d, k, x, a, g []interface{}) {
	xs, ys := make([]float64, len(s)), make(map[string][]float64, 6)
	for i := range s {
		xs[i] = float64(i)
	}
	ys["sto"] = mmio.InterfaceToFloat(s)
	ys["def"] = mmio.InterfaceToFloat(d)
	ys["lag"] = mmio.InterfaceToFloat(k)
	ys["xs"] = mmio.InterfaceToFloat(x)
	ys["aet"] = mmio.InterfaceToFloat(a)
	ys["rch"] = mmio.InterfaceToFloat(g)
	mmio.Line(fp, xs, ys)
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

func sumWriteRMaps(outdir string, xr map[int]int, ds []int, gy, ga, gr, gg, gd, gl []float64, fnstep float64) {
	mgy, mga, mgolf, mgg, mgl := make(map[int]float64, len(xr)), make(map[int]float64, len(xr)), make(map[int]float64, len(xr)), make(map[int]float64, len(xr)), make(map[int]float64, len(xr))
	mron, mroff := make(map[int]float64, len(xr)), make(map[int]float64, len(xr))
	f := 365.24 * 1000. / fnstep
	for c, i := range xr {
		mgy[c] = gy[i] * f
		mga[c] = ga[i] * f
		mgolf[c] = gr[i] * f
		mgg[c] = (gg[i] - gd[i]) * f
		mgl[c] = gl[i] * f
		if ds[i] > -1 {
			mron[ds[i]] += gr[i] * f
		}
	}
	for c := range xr {
		mroff[c] = mgolf[c] - mron[c]
		if mgg[c] < 0. {
			mroff[c] += mgg[c] // exclude runoff from groundwater discharge
		}
		if mroff[c] < 0. {
			mroff[c] = 0. // exclude negative runoff (caused by greater infiltrability)
		}
	}
	// NOTE: wbal = y + ron - (a + g + olf)
	mmio.WriteRMAP(outdir+"precipitation.rmap", mgy, false)
	mmio.WriteRMAP(outdir+"aet.rmap", mga, false)
	mmio.WriteRMAP(outdir+"ro.rmap", mgolf, false)
	mmio.WriteRMAP(outdir+"ron.rmap", mron, false)
	mmio.WriteRMAP(outdir+"ro_gen.rmap", mroff, false)
	mmio.WriteRMAP(outdir+"gwe.rmap", mgg, false)
	mmio.WriteRMAP(outdir+"storage.rmap", mgl, false)
}
