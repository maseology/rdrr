package rdrr

import "github.com/maseology/goHydro/hru"

func (ev *Evaluator) buildRealization(nt int) ([]*realization, [][]int, [][]float64) {
	ns := len(ev.Scids)
	x := make([][]hru.Res, ns)
	rel := make([]*realization, ns)
	mons, monq := make([][]int, ns), [][]float64{}
	for k, cids := range ev.Scids {
		x[k] = make([]hru.Res, len(cids))
		for i, d := range ev.DepSto[k] {
			x[k][i].Cap = d
		}

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
			cds:   ev.Sds[k],
			rte:   ev.Dsws[k],
			eaf:   ev.Eafact,
			dextm: ev.Dext / ev.M[ev.Sgw[k]],
			fnc:   float64(len(cids)),
			fgnc:  ev.Fngwc[ev.Sgw[k]],
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
