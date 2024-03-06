package rdrr

import (
	"fmt"

	"github.com/maseology/goHydro/forcing"
	"github.com/maseology/goHydro/hru"
)

// Evaluate a single run
func (ev *Evaluator) EvaluateSerial(frc *forcing.Forcing, outdirprfx string) (hyd []float64) {
	// prep
	nt, ng, ns := len(frc.T), len(ev.Fngwc), len(ev.Scids)
	qout := make([][]float64, ns)
	x := make([][]hru.Res, ns)
	rel := make([]*realization, ns)
	mons, monq := make([][]int, ns), [][]float64{}
	for k, cids := range ev.Scids {
		qout[k] = make([]float64, nt)
		x[k] = make([]hru.Res, len(cids))
		for i, d := range ev.DepSto[k] {
			x[k][i].Cap = d
		}
		for range ev.Mons[k] {
			mons[k] = append(mons[k], len(monq))
			monq = append(monq, make([]float64, nt))
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
			sds:   ev.Sds[k],
			rte:   ev.Dsws[k],
			// m:    ev.M[ev.Sgw[k]],
			eaf:   ev.Eafact,
			dextm: ev.Dext / ev.M[ev.Sgw[k]],
			fnc:   float64(len(cids)),
			fgnc:  ev.Fngwc[ev.Sgw[k]],
			cmon:  ev.Mons[k],
		}
	}

	// spr, sae, sro, srch := make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc)
	dms, dmsv := make([]float64, ng), make([]float64, ng)
	// cnt := 0
	for j, t := range frc.T {
		fmt.Println(t)
		for i := 0; i < ng; i++ {
			dms[i] += dmsv[i]
			dmsv[i] = 0.
		}
		for k := range ev.Scids {
			m, q, dd := rel[k].rdrr(frc.Ya[k][j], frc.Ea[k][j], dms[ev.Sgw[k]]/ev.M[ev.Sgw[k]], j, k)
			for i, ii := range mons[k] {
				monq[ii][j] = m[i]
			}
			qout[k][j] = q
			func(r SWStopo) {
				if r.Sid < 0 {
					return
				}
				rel[r.Sid].x[r.Cid].Sto += q
			}(rel[k].rte)
			dmsv[ev.Sgw[k]] += dd
		}
		// cnt++
		// if cnt > 10 {
		// 	break
		// }
	}

	hyd = make([]float64, nt)
	for j := range frc.T {
		for k := range ev.Scids {
			hyd[j] += qout[k][j] // / ev.Fnstrm[k]
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

	writeFloats(nil, outdirprfx+"spr.bin", spr)
	writeFloats(nil, outdirprfx+"sae.bin", sae)
	writeFloats(nil, outdirprfx+"sro.bin", sro)
	writeFloats(nil, outdirprfx+"srch.bin", srch)
	writeFloats(nil, outdirprfx+"lsto.bin", lsto)
	writeMons(outdirprfx+"mon.gob", ev.Mons, monq)
	writeFloats(nil, outdirprfx+"hyd.bin", hyd)

	return hyd
}
