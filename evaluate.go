package rdrr

import (
	"fmt"

	"github.com/gosuri/uiprogress"
	"github.com/maseology/rdrr/forcing"
)

// Evaluate a single run, no concurrency
func (ev *Evaluator) EvaluateSerial(frc *forcing.Forcing, outdirprfx string) (hyd []float64) {

	nt, ng := len(frc.T), len(ev.Fngwc)
	rel, mons, monq := ev.buildRealization(nt)
	// qout := make([][]float64, len(ev.Scids))
	// for k := range ev.Scids {
	// 	qout[k] = make([]float64, nt)
	// }

	dms, dmsv := make([]float64, ng), make([]float64, ng)
	hyd = make([]float64, nt)
	uiprogress.Start()
	timestep := make(chan string)
	bar := uiprogress.AddBar(nt).AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return <-timestep
	})
	for j, t := range frc.T {
		// fmt.Println(t)
		timestep <- fmt.Sprint(t)
		for i := 0; i < ng; i++ {
			dms[i] += dmsv[i]
			dmsv[i] = 0.
		}
		for _, inner := range ev.Outer {
			for _, k := range inner {
				m, q, dd := rel[k].rdrr(frc.Ya[k][j], frc.Ea[k][j], dms[ev.Sgw[k]]/ev.M[ev.Sgw[k]], j, k)
				lfact := 1.
				if ev.IsLake[k] {
					lfact = rel[k].fnc // volume adjustment for lakes
				}
				for i, ii := range mons[k] {
					monq[ii][j] = m[i]
				}
				dmsv[ev.Sgw[k]] += dd * lfact
				func(r SWStopo) {
					if r.Sid < 0 {
						hyd[j] = q * lfact
					} else {
						if ev.IsLake[r.Sid] {
							rel[r.Sid].x[0].Sto += q * lfact / rel[r.Sid].fnc
						} else {
							rel[r.Sid].x[r.Cid].Sto += q * lfact
						}
					}
				}(rel[k].rte)
				// qout[k][j] = q
				// func(r SWStopo) {
				// 	if r.Sid < 0 {
				// 		return
				// 	}
				// 	rel[r.Sid].x[r.Cid].Sto += q
				// }(rel[k].rte)
				// dmsv[gid] += dd
			}
		}
		bar.Incr()
		// if j > 10 {
		// 	break
		// }
	}
	close(timestep)
	uiprogress.Stop()

	// hyd = make([]float64, nt)
	// for j := range frc.T {
	// 	for k := range ev.Scids {
	// 		hyd[j] += qout[k][j] // / ev.Fnstrm[k]
	// 	}
	// }

	spr, sae, sro, srch, sgwd, lsto := make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc)
	for k, cids := range ev.Scids {
		if ev.IsLake[k] {
			for _, c := range cids {
				spr[c] = rel[k].spr[0]
				sae[c] = rel[k].sae[0]
				sro[c] = rel[k].sro[0]
				srch[c] = rel[k].srch[0]
				sgwd[c] = rel[k].sgwd[0]
				lsto[c] = rel[k].x[0].Sto
			}
		} else {
			for i, c := range cids {
				spr[c] = rel[k].spr[i]
				sae[c] = rel[k].sae[i]
				sro[c] = rel[k].sro[i]
				srch[c] = rel[k].srch[i]
				sgwd[c] = rel[k].sgwd[i]
				lsto[c] = rel[k].x[i].Sto
			}
		}
	}

	writeFloats(outdirprfx+"spr.bin", spr)
	writeFloats(outdirprfx+"sae.bin", sae)
	writeFloats(outdirprfx+"sro.bin", sro)
	writeFloats(outdirprfx+"srch.bin", srch)
	writeFloats(outdirprfx+"sgwd.bin", sgwd)
	writeFloats(outdirprfx+"lsto.bin", lsto)
	writeFloats(outdirprfx+"hyd.bin", hyd)
	if ev.Mons != nil {
		writeMons(outdirprfx+"mon.gob", ev.Mons, monq)
	}

	return hyd
}
