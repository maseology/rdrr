package rdrr

import (
	"fmt"

	"github.com/gosuri/uiprogress"
	"github.com/maseology/rdrr/forcing"
)

// Evaluate a single run, no concurrency
func (ev *Evaluator) EvaluateSerial(frc *forcing.Forcing, outdirprfx string) (hyd []float64) {
	// prep
	nt, ng := len(frc.T), len(ev.Fngwc)
	rel, rte, sdm, monq, imons := ev.buildRealization(nt, ng)

	uiprogress.Start()
	timestep := make(chan string)
	bar := uiprogress.AddBar(nt).AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return <-timestep
	})

	dms, dmsv := make([]float64, ng), make([]float64, ng)
	hyd = make([]float64, nt)
	for j, t := range frc.T {
		// fmt.Println(t)
		timestep <- fmt.Sprint(t)
		mnt := int(t.Month()) - 1
		for ig := 0; ig < ng; ig++ {
			dms[ig] += dmsv[ig]
			sdm[nt*ig+j] = dms[ig]
			dmsv[ig] = 0.
		}
		for _, inner := range ev.Outer {
			for _, k := range inner {
				relk, gi := rel[k], ev.Sgw[k]
				m, q, dd := relk.rdrr(frc.Ya[k][j], frc.Ea[k][j], dms[gi]/ev.M[gi], mnt, j, k)
				for i, ii := range imons[k] {
					monq[ii*nt+j] = m[i]
				}
				dmsv[gi] += dd
				if rte[k] == nil {
					hyd[j] = q
				} else {
					rte[k].Sto += q
				}
				// func(r SWStopo) {
				// 	if r.Sid < 0 {
				// 		hyd[j] = q
				// 	} else {
				// 		rel[r.Sid].x[r.Cid].Sto += q
				// 	}
				// }(relk.rte)
				// qout[k][j] = q
				// func(r SWStopo) {
				// 	if r.Sid < 0 {
				// 		return
				// 	}
				// 	rel[r.Sid].x[r.Cid].Sto += q
				// }(relk.rte)
				// dmsv[gid] += dd
			}
		}
		bar.Incr()
		// if j > 10 {
		// 	break
		// }
	}
	close(timestep)
	uiprogress.Stop()

	ev.saveToBins(rel, sdm, monq, hyd, nt, outdirprfx)

	return hyd
}
