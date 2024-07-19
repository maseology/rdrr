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
	rel, sdm, monq := ev.buildRealization(nt, ng)

	// nt, ng, ns := len(frc.T), len(ev.Fngwc), len(ev.Scids)
	// // qout := make([][]float64, ns)
	// x := make([][]hru.Res, ns)
	// rel := make([]*realization, ns)
	// mons, monq := make([][]int, ns), [][]float64{}
	// for k, cids := range ev.Scids {
	// 	// qout[k] = make([]float64, nt)
	// 	x[k] = make([]hru.Res, len(cids))
	// 	for i, d := range ev.DepSto[k] {
	// 		x[k][i].Cap = d
	// 	}

	// 	rel[k] = &realization{
	// 		x:     x[k],
	// 		drel:  ev.Drel[k],
	// 		bo:    ev.Bo[k],
	// 		finf:  ev.Finf[k],
	// 		fcasc: ev.Fcasc[k],
	// 		spr:   make([]float64, len(cids)),
	// 		sae:   make([]float64, len(cids)),
	// 		sro:   make([]float64, len(cids)),
	// 		srch:  make([]float64, len(cids)),
	// 		sgwd:  make([]float64, len(cids)),
	// 		cids:  cids,
	// 		cds:   ev.Sds[k],
	// 		rte:   ev.Dsws[k],
	// 		// m:    ev.M[ev.Sgw[k]],
	// 		eaf:   ev.Eafact,
	// 		dextm: ev.Dext / ev.M[ev.Sgw[k]],
	// 		fnc:   float64(len(cids)),
	// 		fgnc:  ev.Fngwc[ev.Sgw[k]],
	// 		// cmon:  ev.Mons[k],
	// 	}

	// 	if ev.Mons != nil {
	// 		for range ev.Mons[k] {
	// 			mons[k] = append(mons[k], len(monq))
	// 			monq = append(monq, make([]float64, nt))
	// 		}
	// 		rel[k].cmon = ev.Mons[k]
	// 	}
	// }

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
			sdm[12*ig+j] = dms[ig]
			dmsv[ig] = 0.
		}
		for _, inner := range ev.Outer {
			for _, k := range inner {
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
		if j > 10 {
			break
		}
	}
	close(timestep)
	uiprogress.Stop()

	ev.saveToBins(rel, sdm, monq, hyd, nt, outdirprfx)

	return hyd
}
