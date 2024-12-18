package rdrr

import (
	"sync"

	"github.com/maseology/rdrr/forcing"
)

func (ev *Evaluator) Evaluate(frc *forcing.Forcing, outdirprfx string) (hyd []float64) {

	// prep
	nt, ng := len(frc.T), len(ev.Fngwc)
	rel, sdm, monq := ev.buildRealization(nt, ng)

	var wg sync.WaitGroup
	dms, dmsv := make([]float64, ng), make([]float64, ng)
	hyd = make([]float64, nt)
	for j, t := range frc.T {
		mnt := int(t.Month()) - 1
		for ig := 0; ig < ng; ig++ {
			dms[ig] += dmsv[ig]
			sdm[nt*ig+j] = dms[ig]
			dmsv[ig] = 0.
		}
		for _, inner := range ev.Outer {
			wg.Add(len(inner))
			for _, k := range inner {
				go func(k int) {
					relk, gi := rel[k], ev.Sgw[k]
					m, q, dd := relk.rdrr(frc.Ya[k][j], frc.Ea[k][j], dms[gi]/ev.M[gi], mnt, j, k)
					for i, ii := range relk.imon {
						monq[ii*nt+j] = m[i]
					}
					dmsv[gi] += dd
					if relk.rte == nil {
						hyd[j] = q
					} else {
						relk.rte.Sto += q
					}
					wg.Done()
				}(k)
			}
			wg.Wait()
		}
	}

	if len(outdirprfx) > 0 {
		ev.saveToBins(rel, sdm, monq, hyd, nt, outdirprfx)
	}

	return hyd
}
