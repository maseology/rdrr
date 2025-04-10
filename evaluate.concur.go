package rdrr

import (
	"sync"

	"github.com/maseology/rdrr/forcing"
)

func (ev *Evaluator) Evaluate(frc *forcing.Forcing, outdirprfx string) (hyd []float64) {

	// prep
	nt, ng := len(frc.T), len(ev.Fngwc)
	rel, sdm := ev.buildBasin(nt, ng, len(outdirprfx) > 0)
	// rte := ev.buildRoute()
	monq := make([]float64, nt*ev.Nm)

	var wg sync.WaitGroup
	dms, dmsv := make([]float64, ng), make([]float64, ng)
	hyd = make([]float64, nt)
	stage := make([]float64, len(rel))

	for j, t := range frc.T {
		mnt := int(t.Month()) - 1
		for ig := range ng {
			dms[ig] += dmsv[ig]
			sdm[nt*ig+j] = dms[ig]
			dmsv[ig] = 0.
		}

		// update cell state
		k := make(chan int)
		done := make(chan any)
		wg.Add(nthrd)
		for range nthrd {
			go func(done <-chan any, kchan <-chan int) {
				for {
					select {
					case <-done:
						wg.Done()
						return
					case k := <-kchan:
						relk, gi := rel[k], ev.Sgw[k]
						q, dd := relk.rdrr(frc.Ya[k][j], frc.Ea[k][j], dms[gi]/ev.M[gi], mnt, j, k)
						dmsv[gi] += dd
						stage[k] = q
					}
				}
			}(done, k)
		}

		for i := range rel {
			k <- i
		}
		close(done)
		wg.Wait()
		close(k)

		// // route SWSs
		// for _, inner := range ev.Outer {
		// 	wg.Add(len(inner))
		// 	for _, k := range inner {
		// 		go func(k int) {
		// 			q := rte[k].conv.Update(stage[k])
		// 			if q > 0 && rte[k].sds > -1 {
		// 				stage[rte[k].sds] += q
		// 			} else {
		// 				hyd[j] += q
		// 			}
		// 			if i := ev.Smon[k]; i >= 0 {
		// 				monq[i*nt+j] = q
		// 			}
		// 			wg.Done()
		// 		}(k)
		// 	}
		// 	wg.Wait()
		// }
		// if j > 10 {
		// 	break
		// }
	}

	if len(outdirprfx) > 0 {
		ev.saveToBins(rel, sdm, monq, hyd, outdirprfx)
	}

	return hyd
}

// func (ev *Evaluator) Evaluate(frc *forcing.Forcing, outdirprfx string, collectGrids bool) (hyd []float64) {

// 	// prep
// 	nt, ng := len(frc.T), len(ev.Fngwc)
// 	rel, rte, sdm, monq, imons := ev.buildRealization(nt, ng, collectGrids)

// 	var wg sync.WaitGroup
// 	dms, dmsv := make([]float64, ng), make([]float64, ng)
// 	hyd = make([]float64, nt)
// 	for j, t := range frc.T {
// 		mnt := int(t.Month()) - 1
// 		for ig := 0; ig < ng; ig++ {
// 			dms[ig] += dmsv[ig]
// 			sdm[nt*ig+j] = dms[ig]
// 			dmsv[ig] = 0.
// 		}
// 		for _, inner := range ev.Outer {
// 			wg.Add(len(inner))
// 			for _, k := range inner {
// 				go func(k int) {
// 					relk, gi := rel[k], ev.Sgw[k]
// 					m, q, dd := relk.rdrr(frc.Ya[k][j], frc.Ea[k][j], dms[gi]/ev.M[gi], mnt, j, k)
// 					for i, ii := range imons[k] {
// 						monq[ii*nt+j] = m[i]
// 					}
// 					dmsv[gi] += dd
// 					if rte[k] == nil {
// 						hyd[j] = q
// 					} else {
// 						rte[k].Sto += q
// 					}
// 					wg.Done()
// 				}(k)
// 			}
// 			wg.Wait()
// 		}
// 	}

// 	if len(outdirprfx) > 0 {
// 		ev.saveToBins(rel, sdm, monq, hyd, nt, outdirprfx)
// 	}

// 	return hyd
// }
