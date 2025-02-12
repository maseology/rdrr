package rdrr

import (
	"fmt"

	"github.com/maseology/goHydro/grid"
)

type Evaluator struct {
	Outer, Saids, Sads, Mons      [][]int // Incs, Dwnas
	Dsws                          []SWStopo
	Drel, Bo, Fcasc, Finf, DepSto [][]float64
	Sgw                           []int
	M, Fngwc                      []float64 // , Fnstrm
	Eafact, Dext                  float64   // open water evaporation coefficient, extinction depth
	Nc                            int
	// IsLake                        []bool
}

func (ev *Evaluator) CheckAndPrint(gd *grid.Definition, cids, igw []int, chkdirprfx string, crop bool) {

	var gdcrp *grid.Definition
	xr := make(map[int]int)
	if crop {
		gdcrp, xr = gd.CropToActives()
	} else {
		gdcrp = gd
		for _, c := range gd.Sactives {
			xr[c] = c
		}
	}

	// output
	sgw, sads := gdcrp.NullInt32(-9999), gdcrp.NullInt32(-9999)
	drel, bo, fcasc, finf, dsto, m := gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.), gdcrp.NullArray(-9999.)
	for k, saids := range ev.Saids {
		for i, ac := range saids {
			var c int
			if x, ok := xr[cids[ac]]; ok {
				c = x
			} else {
				fmt.Println(ac, cids[ac])
				panic("111")
			}
			// c := xr[cids[ac]]
			drel[c] = ev.Drel[k][i]
			bo[c] = ev.Bo[k][i]
			fcasc[c] = ev.Fcasc[k][i]
			finf[c] = ev.Finf[k][i]
			dsto[c] = ev.DepSto[k][i]
			m[c] = ev.M[igw[ac]]
			sgw[c] = int32(ev.Sgw[k])
			sads[c] = int32(ev.Sads[k][i])
		}
	}

	writeInts(gdcrp, chkdirprfx+"evaluator.sgw.bil", sgw)          // groundwater index, now projected to sws
	writeInts(gdcrp, chkdirprfx+"evaluator.sads.bil", sads)        // downslope cell ID by SWS, <0 is routed to down-SWS
	writeFloats32(gdcrp, chkdirprfx+"evaluator.drel.bil", drel)    // groundwater deficit relative to the regional mean (deltaD)
	writeFloats32(gdcrp, chkdirprfx+"evaluator.bo.bil", bo)        // groundwater flux to surface/channels
	writeFloats32(gdcrp, chkdirprfx+"evaluator.fcasc.bil", fcasc)  // fraction of excess storage to runoff
	writeFloats32(gdcrp, chkdirprfx+"evaluator.finf.bil", finf)    // fraction of excess storage to infiltrate assuming a falling head through a unit length per timestep
	writeFloats32(gdcrp, chkdirprfx+"evaluator.depsto.bil", dsto)  // depression storage
	writeFloats32(gdcrp, chkdirprfx+"evaluator.TOPMODEL-m.bil", m) // TOPMODEL parameter m
}
