package rdrr

import "github.com/maseology/goHydro/hru"

func (ev *Evaluator) buildBasin(nt, ng int, collectGrids bool) ([]*basin, []float64) {
	ns := len(ev.Sais)
	rel := make([]*basin, ns)

	for k, aids := range ev.Sais {
		x := make([]hru.Res, len(aids))
		for i, d := range ev.DepSto[k] {
			x[i].Cap = d
		}

		var col *basinCollect
		if collectGrids {
			col = &basinCollect{
				spr:  make([]float64, len(aids)*12),
				sae:  make([]float64, len(aids)*12),
				sro:  make([]float64, len(aids)*12),
				srch: make([]float64, len(aids)*12),
			}
		}

		sads := make([]int, len(aids))
		for i := range aids {
			if ev.IsStrm[k][i] {
				sads[i] = -1
			} else {
				sads[i] = ev.Sads[k][i]
			}
		}

		rel[k] = &basin{
			x:     x,
			drel:  ev.Drel[k],
			bo:    ev.Bo[k],
			finf:  ev.Finf[k],
			fcasc: ev.Fcasc[k],
			cids:  aids,
			ads:   sads,
			eaf:   ev.Eafact,
			dextm: ev.Dext / ev.M[ev.Sgw[k]],
			fnc:   float64(len(aids)),
			fgnc:  ev.Fngwc[ev.Sgw[k]],
			nc:    len(aids),
			coll:  col,
		}
	}

	sdm := make([]float64, nt*ng)
	return rel, sdm
}
