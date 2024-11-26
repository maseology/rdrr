package rdrr

import "github.com/maseology/goHydro/grid"

type Evaluator struct {
	Outer, Scids, Sds, Mons       [][]int // Incs, Dwnas
	Dsws                          []SWStopo
	Drel, Bo, Fcasc, Finf, DepSto [][]float64
	Sgw                           []int
	M, Fngwc                      []float64 // , Fnstrm
	Eafact, Dext                  float64   // open water evaporation coefficient, extinction depth
	Nc                            int
	// IsLake                        []bool
}

func (ev *Evaluator) CheckAndPrint(gd *grid.Definition, cids, igw []int, chkdirprfx string) {

	gdcrp, xr := gd.CropToActives()

	// output
	sgw, sds := gdcrp.NullInt32(-9999), gdcrp.NullInt32(-9999)
	drel, bo, fcasc, finf, dsto, m := gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.)
	for k, scids := range ev.Scids {
		for i, sc := range scids {
			c := xr[cids[sc]]
			drel[c] = ev.Drel[k][i]
			bo[c] = ev.Bo[k][i]
			fcasc[c] = ev.Fcasc[k][i]
			finf[c] = ev.Finf[k][i]
			dsto[c] = ev.DepSto[k][i]
			m[c] = ev.M[igw[sc]]
			sgw[c] = int32(ev.Sgw[k])
			sds[c] = int32(ev.Sds[k][i])
		}
	}

	writeInts(gdcrp, chkdirprfx+"evaluator.sgw.bil", sgw)          // groundwater index, now projected to sws
	writeInts(gdcrp, chkdirprfx+"evaluator.sds.bil", sds)          // downslope cell ID by SWS, <0 is routed to down-SWS
	writeFloats32(gdcrp, chkdirprfx+"evaluator.drel.bil", drel)    // groundwater deficit relative to the regional mean (deltaD)
	writeFloats32(gdcrp, chkdirprfx+"evaluator.bo.bil", bo)        // groundwater flux to surface/channels
	writeFloats32(gdcrp, chkdirprfx+"evaluator.fcasc.bil", fcasc)  // fraction of excess storage to runoff
	writeFloats32(gdcrp, chkdirprfx+"evaluator.finf.bil", finf)    // fraction of excess storage to infiltrate assuming a falling head through a unit length per timestep
	writeFloats32(gdcrp, chkdirprfx+"evaluator.depsto.bil", dsto)  // depression storage
	writeFloats32(gdcrp, chkdirprfx+"evaluator.TOPMODEL-m.bil", m) // TOPMODEL parameter m
}
