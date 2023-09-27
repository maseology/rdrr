package rdrr

import (
	"sync"

	"github.com/maseology/goHydro/forcing"
	"github.com/maseology/goHydro/hru"
)

func (ev *Evaluator) Evaluate(frc *forcing.Forcing, outdirprfx string) (hyd []float64) {

	// done := make(chan interface{})
	// defer close(done)

	// prep
	ng, ns := len(ev.Fngwc), len(ev.Scids)
	x := make([][]hru.Res, ns)
	rel := make([]*realization, ns)
	// mon := make([]map[int][]float64, ns)
	for k, cids := range ev.Scids {
		x[k] = make([]hru.Res, len(cids))
		for i, d := range ev.DepSto[k] {
			x[k][i].Cap = d
		}
		// for _, m := range ev.Mons[k] {
		// 	mon[k][m] = make([]float64, nt)
		// }
		rel[k] = &realization{
			x:     x[k],
			drel:  ev.Drel[k],
			bo:    ev.Bo[k],
			finf:  ev.Finf[k],
			fcasc: ev.Fcasc[k],
			spr:   make([]float64, len(cids)),
			sae:   make([]float64, len(cids)),
			sro:   make([]float64, len(cids)),
			srch:  make([]float64, len(cids)),
			cids:  cids,
			sds:   ev.Sds[k],
			// m:    ev.M[ev.Sgw[k]],
			eaf:   ev.Eafact,
			dextm: ev.Dext / ev.M[ev.Sgw[k]],
			fnc:   float64(len(cids)),
			fgnc:  ev.Fngwc[ev.Sgw[k]],
		}
	}

	var wg sync.WaitGroup
	dms, dmsv := make([]float64, ng), make([]float64, ng)
	for j := range frc.T {
		// fmt.Println(t)
		for i := 0; i < ng; i++ {
			dms[i] += dmsv[i]
			dmsv[i] = 0.
		}
		for _, inner := range ev.Outer {
			wg.Add(len(inner))
			for _, k := range inner {
				go func(k int) {
					_, dd := rel[k].rdrr(frc.Ya[k][j], frc.Ea[k][j], dms[ev.Sgw[k]]/ev.M[ev.Sgw[k]], j, k)
					dmsv[ev.Sgw[k]] += dd
					wg.Done()
				}(k)
			}
			wg.Wait()
		}
	}

	spr, sae, sro, srch, lsto := make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc)
	for k, cids := range ev.Scids {
		for i, c := range cids {
			spr[c] = rel[k].spr[i]
			sae[c] = rel[k].sae[i]
			sro[c] = rel[k].sro[i]
			srch[c] = rel[k].srch[i]
			lsto[c] = rel[k].x[i].Sto
		}
	}

	writeFloats(outdirprfx+"spr.bin", spr)
	writeFloats(outdirprfx+"sae.bin", sae)
	writeFloats(outdirprfx+"sro.bin", sro)
	writeFloats(outdirprfx+"srch.bin", srch)
	writeFloats(outdirprfx+"lsto.bin", lsto)
	// writeFloats(outdirprfx+"hyd.bin", hyd)
	return hyd
}

// // pipeline/workers
// func evalstream(done <-chan interface{}, rel <-chan realization, nwrkrs int) <-chan result {
// 	evalstream := make(chan result)
// 	for i := 0; i < nwrkrs; i++ {
// 		go func(i int) {
// 			// defer close(evalstream)
// 			for {
// 				select {
// 				case <-done:
// 					return
// 				case r := <-rel:
// 					evalstream <- r.rdrr()
// 				}
// 			}
// 			// for r := range rel {
// 			// 	select {
// 			// 	case <-done:
// 			// 		return
// 			// 	default:
// 			// 		evalstream <- r.rdrr()
// 			// 	}
// 			// }
// 		}(i)
// 	}
// 	return evalstream
// }
