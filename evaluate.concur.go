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
	ng, ns, nt := len(ev.Fngwc), len(ev.Scids), len(frc.T)
	x := make([][]hru.Res, ns)
	rel := make([]*realization, ns)
	mons, monq := make([][]int, ns), [][]float64{}
	for k, cids := range ev.Scids {
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
					m, q, dd := rel[k].rdrr(frc.Ya[k][j], frc.Ea[k][j], dms[ev.Sgw[k]]/ev.M[ev.Sgw[k]], j, k)
					for i, ii := range mons[k] {
						monq[ii][j] = m[i]
					}
					dmsv[ev.Sgw[k]] += dd
					func(r SWStopo) {
						if r.Sid < 0 {
							return
						}
						rel[r.Sid].x[r.Cid].Sto += q
					}(rel[k].rte)
					_ = q
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

	writeFloats(nil, outdirprfx+"spr.bin", spr)
	writeFloats(nil, outdirprfx+"sae.bin", sae)
	writeFloats(nil, outdirprfx+"sro.bin", sro)
	writeFloats(nil, outdirprfx+"srch.bin", srch)
	writeFloats(nil, outdirprfx+"lsto.bin", lsto)
	writeMons(outdirprfx+"mon.gob", ev.Mons, monq)
	// writeFloats(outdirprfx+"hyd.bin", hyd)
	return hyd
}
