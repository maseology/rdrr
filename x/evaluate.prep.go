package rdrr

import "time"

func (ev *Evaluator) prep(ts []time.Time) ([][][]float64, [][]float64, []int) {
	nt, ng, ns := len(ts), len(ev.Fngwc), len(ev.Scids)

	deldsv := make([][]float64, ng)
	for i := 0; i < ng; i++ {
		deldsv[i] = make([]float64, nt)
	}

	xsv := make([][][]float64, ns)
	for is, v := range ev.Incs {
		xsv[is] = make([][]float64, len(v))
		for i := range v {
			xsv[is][i] = make([]float64, nt)
		}
	}

	mth := func() []int {
		o := make([]int, nt)
		for i, t := range ts {
			o[i] = int(t.Month()) - 1
		}
		return o
	}()

	return xsv, deldsv, mth
}
