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
	mons := make([][]float64, ns)
	for k, cids := range ev.Scids {
		qout[k] = make([]float64, nt)
		x[k] = make([]hru.Res, len(cids))
		for i, d := range ev.DepSto[k] {
			x[k][i].Cap = d
		}
		if ev.Mons[k] >= 0 {
			mons[k] = make([]float64, nt)
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
	for j, t := range frc.T {
		fmt.Println(t)
		for i := 0; i < ng; i++ {
			dms[i] += dmsv[i]
			dmsv[i] = 0.
		}
		for k := range ev.Scids {
			q, m, dd := rel[k].rdrr(frc.Ya[k][j], frc.Ea[k][j], dms[ev.Sgw[k]]/ev.M[ev.Sgw[k]], j, k)
			if ev.Mons[k] >= 0 {
				mons[k][j] = m
			}
			qout[k][j] = q
			dmsv[ev.Sgw[k]] += dd
		}
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

	writeFloats(outdirprfx+"spr.bin", spr)
	writeFloats(outdirprfx+"sae.bin", sae)
	writeFloats(outdirprfx+"sro.bin", sro)
	writeFloats(outdirprfx+"srch.bin", srch)
	writeFloats(outdirprfx+"lsto.bin", lsto)
	writeMons(outdirprfx+"mon.gob", ev.Mons, mons)
	writeFloats(outdirprfx+"hyd.bin", hyd)

	return hyd
}
