package rdrr

import (
	"github.com/maseology/goHydro/grid"
)

type Evaluator struct {
	Outer, Scids, Sds, Incs, Dwnas, Mons [][]int
	Drel, Bo, Fcasc, Finf, DepSto        [][]float64
	Sgw                                  []int
	M, Fngwc                             []float64
	Dext, Eafact                         float64
}

func (ev *Evaluator) Checkandprint(gd *grid.Definition, cids, igw []int, chkdirprfx string) {

	// output
	sgw := gd.NullInt32(-9999)
	drel, bo, fcasc, finf, dsto, m := gd.NullArray(-9999.), gd.NullArray(-9999.), gd.NullArray(-9999.), gd.NullArray(-9999.), gd.NullArray(-9999.), gd.NullArray(-9999.)
	for k, scids := range ev.Scids {
		for i, sc := range scids {
			c := cids[sc]
			drel[c] = ev.Drel[k][i]
			bo[c] = ev.Bo[k][i]
			fcasc[c] = ev.Fcasc[k][i]
			finf[c] = ev.Finf[k][i]
			dsto[c] = ev.DepSto[k][i]
			m[c] = ev.M[igw[sc]]
			sgw[c] = int32(ev.Sgw[k])
		}
	}

	writeInts(chkdirprfx+"evaluator.sgw.indx", sgw)
	writeFloats(chkdirprfx+"evaluator.drel.bil", drel)
	writeFloats(chkdirprfx+"evaluator.bo.bil", bo)
	writeFloats(chkdirprfx+"evaluator.fcasc.bil", fcasc)
	writeFloats(chkdirprfx+"evaluator.finf.bil", finf)
	writeFloats(chkdirprfx+"evaluator.depsto.bil", dsto)
	writeFloats(chkdirprfx+"evaluator.TOPMODEL-m.bil", m)
}
