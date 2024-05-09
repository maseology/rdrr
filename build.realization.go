package rdrr

import "github.com/maseology/goHydro/hru"

func (ev *Evaluator) buildRealization(nt int) ([]*realization, [][]int, [][]float64) {
	ns := len(ev.Scids)
	x := make([][]hru.Res, ns)
	rel := make([]*realization, ns)
	mons, monq := make([][]int, ns), [][]float64{}
	for k, scids := range ev.Scids {
		drel := ev.Drel[k]
		bo := ev.Bo[k]
		finf := ev.Finf[k]
		fcasc := ev.Fcasc[k]
		cds := ev.Sds[k]
		ncids := scids
		if ev.IsLake[k] {
			fnc := float64(len(scids))
			x[k] = make([]hru.Res, 1)
			fcasc = []float64{ev.Fcasc[k][len(scids)-1] / fnc}
			ncids = []int{scids[len(scids)-1]}
			cds = []int{-1}
			drel, bo, finf = func() (_, _, _ []float64) {
				d, b, fi := 0., 0., 0.
				for i := range scids {
					d += ev.Drel[k][i]
					b += ev.Bo[k][i]
				}
				return []float64{d / fnc}, []float64{b / fnc}, []float64{fi / fnc}
			}()
			func() {
				depsto := 0.
				for _, d := range ev.DepSto[k] {
					depsto += d
				}
				x[k][0].Cap = depsto / float64(len(ev.DepSto[k]))
			}()
		} else {
			x[k] = make([]hru.Res, len(scids))
			for i, d := range ev.DepSto[k] {
				x[k][i].Cap = d
			}
		}

		rel[k] = &realization{
			x: x[k],
			// drel:  ev.Drel[k],
			// bo:    ev.Bo[k],
			// finf:  ev.Finf[k],
			// fcasc: ev.Fcasc[k],
			// cds:   ev.Sds[k],
			// cids:  cids,
			drel:  drel,
			bo:    bo,
			finf:  finf,
			fcasc: fcasc,
			cds:   cds,
			cids:  ncids,
			spr:   make([]float64, len(scids)),
			sae:   make([]float64, len(scids)),
			sro:   make([]float64, len(scids)),
			srch:  make([]float64, len(scids)),
			sgwd:  make([]float64, len(scids)),
			rte:   ev.Dsws[k],
			// m:    ev.M[ev.Sgw[k]],
			eaf:   ev.Eafact,
			dextm: ev.Dext / ev.M[ev.Sgw[k]],
			fnc:   float64(len(scids)),
			fgnc:  ev.Fngwc[ev.Sgw[k]],
			// cmon:  ev.Mons[k],
		}

		if ev.Mons != nil {
			for range ev.Mons[k] {
				mons[k] = append(mons[k], len(monq))
				monq = append(monq, make([]float64, nt))
			}
			rel[k].cmon = ev.Mons[k]
		}
	}
	return rel, mons, monq
}

// // prep
// ng, ns, nt := len(ev.Fngwc), len(ev.Scids), len(frc.T)
// x := make([][]hru.Res, ns)
// rel := make([]*realization, ns)
// mons, monq := make([][]int, ns), [][]float64{}
// for k, cids := range ev.Scids {
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
