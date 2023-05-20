package rdrr

import (
	"fmt"

	"github.com/maseology/goHydro/hru"
)

// Evaluate a single run
func (ev *Evaluator) Evaluate(frc *Forcing, nwrkrs int, outdirprfx string) (hyd []float64) {
	return ev.evaluate(frc, nwrkrs, outdirprfx)
}

func (ev *Evaluator) evaluate(frc *Forcing, nwrkrs int, outdirprfx string) []float64 {

	// prep
	nt, ng, ns := len(frc.T), len(ev.Fngwc), len(ev.Scids)
	qout := make([][]float64, ns)
	x := make([][]hru.Res, ns)
	rel := make([]*realization, ns)
	// mon := make([]map[int][]float64, ns)
	for k, cids := range ev.Scids {
		qout[k] = make([]float64, nt)
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

	// spr, sae, sro, srch := make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc)
	dms, dmsv := make([]float64, ng), make([]float64, ng)
	for j, t := range frc.T {
		fmt.Println(t)
		for i := 0; i < ng; i++ {
			dms[i] += dmsv[i]
			dmsv[i] = 0.
		}
		for k := range ev.Scids {
			q, dd := rel[k].eval(frc.Ya[k][j], frc.Ea[k][j], dms[ev.Sgw[k]]/ev.M[ev.Sgw[k]], j, k)
			qout[k][j] = q
			dmsv[ev.Sgw[k]] += dd
		}
	}

	hyd := make([]float64, nt)
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
	writeFloats(outdirprfx+"hyd.bin", hyd)

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

// func (ev *Evaluator) evaluateOLD(frc *Forcing, nc, nwrkrs int, outdirprfx string, signal chan<- bool) (hyd []float64) {

// 	done := make(chan interface{})
// 	defer close(done)
// 	rel := make(chan realization, nwrkrs)
// 	res := evalstream(done, rel, nwrkrs)

// 	nt, ng := len(frc.T), len(ev.Fngwc)
// 	spr, sae, sro, srch, lsto := make([]float64, nc), make([]float64, nc), make([]float64, nc), make([]float64, nc), make([]float64, nc)
// 	// xsv, deldsv, mth := ev.prep(frc.T)
// 	dm := make([]float64, ng)

// 	// rel := make(chan realization, nwrkrs)
// 	prcd := make(chan bool)
// 	go func() {
// 		for _, inner := range ev.Outer {
// 			for _, is := range inner {
// 				rr := realization{
// 					i:      is,
// 					ts:     mth,
// 					ds:     ev.Sds[is],
// 					incs:   ev.Incs[is],
// 					mons:   ev.Mons[is],
// 					ins:    xsv[is],
// 					c:      ev.Scids[is],
// 					ya:     frc.Ya[is],
// 					ea:     frc.Ea[is],
// 					deld:   deldsv[ev.Sgw[is]],
// 					drel:   ev.Drel[is],
// 					bo:     ev.Bo[is],
// 					finf:   ev.Finf[is],
// 					depsto: ev.DepSto[is],
// 					fcasc:  ev.Fcasc[is],
// 					fngwc:  ev.Fngwc[ev.Sgw[is]],
// 					m:      ev.M[ev.Sgw[is]],
// 					// dext:   ev.Dext,
// 					eafact: ev.Eafact,
// 					d0:     dm[ev.Sgw[is]],
// 				}
// 				// if ii == 0 {
// 				// 	rr.Steady(200., 10) // spin-up
// 				// }
// 				rel <- rr
// 				// fmt.Printf("sent %d\n", is)
// 			}
// 			<-prcd
// 		}
// 	}()

// 	// r := evalstream(done, rel, nwrkrs)
// 	var wg sync.WaitGroup
// 	// var hyd []float64
// 	// first := true
// 	for _, inner := range ev.Outer {
// 		wg.Add(len(inner))
// 		// if first {
// 		// 	if len(inner) == 1 {
// 		// 		signal <- true
// 		// 		first = false
// 		// 	}
// 		// }
// 		dinner := make([][]float64, ng)
// 		for i := 0; i < ng; i++ {
// 			dinner[i] = make([]float64, nt)
// 		}
// 		for range inner {
// 			go func() {
// 				r := <-res
// 				is := r.i
// 				// fmt.Printf("received %d\n", is)

// 				dm[ev.Sgw[is]] = r.dmlast // setting last d of last round to initial d, this should help spinup issues
// 				dsc := ev.Dwnas[is]
// 				if dsc != nil {
// 					xsv[dsc[0]][dsc[1]] = r.q // outlet of current sws to downslope sws
// 				} else {
// 					hyd = r.q
// 				}

// 				ig := ev.Sgw[is]
// 				for j := 0; j < nt; j++ {
// 					dinner[ig][j] += r.d[j] // concurrent safe: this operation modifies the members of the slice, not modifying the slice itself
// 					// deldsv[ig][j] = r.d[j]
// 				}

// 				for i, c := range ev.Scids[is] {
// 					for j := 0; j < nbins; j++ {
// 						spr[c] += r.pr[i][j]
// 						sae[c] += r.ae[i][j]
// 						sro[c] += r.ro[i][j]
// 						srch[c] += r.rch[i][j]
// 					}
// 					lsto[c] = r.s[i]
// 				}

// 				for _, c := range ev.Mons[is] {
// 					if a, ok := r.mons[c]; ok {
// 						writeFloats(outdirprfx+fmt.Sprintf("mon.%d.%d.bin", is, c), a)
// 					} else {
// 						panic("wtf")
// 					}
// 				}

// 				wg.Done()
// 			}()
// 		}
// 		wg.Wait()

// 		// update state
// 		for ig := range ev.Fngwc {
// 			for j := 0; j < nt; j++ {
// 				deldsv[ig][j] = dinner[ig][j]
// 			}
// 		}

// 		prcd <- true
// 	}

// 	writeFloats(outdirprfx+"spr.bin", spr)
// 	writeFloats(outdirprfx+"sae.bin", sae)
// 	writeFloats(outdirprfx+"sro.bin", sro)
// 	writeFloats(outdirprfx+"srch.bin", srch)
// 	writeFloats(outdirprfx+"lsto.bin", lsto)
// 	writeFloats(outdirprfx+"hyd.bin", hyd)
// 	return hyd
// }
