package rdrr

import "github.com/maseology/goHydro/hru"

func (ev *Evaluator) buildRealization(nt, ng int) ([]*realization, []*hru.Res, []float64, []float64, [][]int) {
	ns, nmon := len(ev.Scids), 0
	rel := make([]*realization, ns)
	rte := make([]*hru.Res, ns)
	imons := make([][]int, ns)

	for k, cids := range ev.Scids {
		x := make([]hru.Res, len(cids))
		for i, d := range ev.DepSto[k] {
			x[i].Cap = d
		}
		// x := make([]hru.Tank, len(cids))
		// for i, d := range ev.DepSto[k] {
		// 	x[i].Dz = d
		// 	x[i].A = ev.Fcasc[k][i]
		// }

		rel[k] = &realization{
			x:     x,
			drel:  ev.Drel[k],
			bo:    ev.Bo[k],
			finf:  ev.Finf[k],
			fcasc: ev.Fcasc[k],
			// spr:   make([]float64, len(cids)*12),
			// sae:   make([]float64, len(cids)*12),
			// sro:   make([]float64, len(cids)*12),
			// srch:  make([]float64, len(cids)*12),
			cids:  cids,
			cds:   ev.Sds[k],
			eaf:   ev.Eafact,
			dextm: ev.Dext / ev.M[ev.Sgw[k]],
			fnc:   float64(len(cids)),
			fgnc:  ev.Fngwc[ev.Sgw[k]],
			nc:    len(cids),
		}

		if ev.Mons != nil {
			if len(ev.Mons[k]) > 0 {
				imon := make([]int, len(ev.Mons[k]))
				rel[k].cmon = make([]int, len(ev.Mons[k]))
				for i, c := range ev.Mons[k] {
					imon[i] = nmon
					rel[k].cmon[i] = c // cell id of monitor
					nmon++
				}
				imons[k] = imon // cross reference to monq
			}
		}
	}

	// set up routing
	for k := range ev.Scids {
		r := ev.Dsws[k]
		if r.Sid < 0 {
			rte[k] = nil
		} else {
			rte[k] = &rel[r.Sid].x[r.Cid]
		}
	}

	sdm, monq := make([]float64, nt*ng), make([]float64, nt*nmon)
	return rel, rte, sdm, monq, imons
}
