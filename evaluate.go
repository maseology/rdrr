package rdrr

import (
	"fmt"
	"sync"
)

// Evaluate a single run
func (ev *Evaluator) Evaluate(frc *Forcing, nc, nwrkrs int, outdirprfx string) (hyd []float64) {
	ev.evaluate(frc, nc, nwrkrs, outdirprfx, nil)
	return nil
}

// pipeline/workers
func evalstream(done <-chan interface{}, rel <-chan realization, nwrkrs int) <-chan result {
	evalstream := make(chan result)
	for i := 0; i < nwrkrs; i++ {
		go func(i int) {
			// defer close(evalstream)
			for {
				select {
				case <-done:
					return
				case r := <-rel:
					evalstream <- r.rdrr()
				}
			}
			// for r := range rel {
			// 	select {
			// 	case <-done:
			// 		return
			// 	default:
			// 		evalstream <- r.rdrr()
			// 	}
			// }
		}(i)
	}
	return evalstream
}

func (ev *Evaluator) evaluate(frc *Forcing, nc, nwrkrs int, outdirprfx string, signal chan<- bool) (hyd []float64) {

	done := make(chan interface{})
	defer close(done)
	rel := make(chan realization, nwrkrs)
	res := evalstream(done, rel, nwrkrs)

	nt, ng, ns := len(frc.T), len(ev.Fngwc), len(ev.Scids)
	sae, sro, srch := make([]float64, nc), make([]float64, nc), make([]float64, nc)
	deldsv, dm := make([][]float64, ng), make([]float64, ng)

	for i := 0; i < ng; i++ {
		deldsv[i] = make([]float64, nt) // INITIAL CONDITIONS: saturated gw
	}

	xsv := make([][][]float64, ns)
	for is, v := range ev.Incs {
		xsv[is] = make([][]float64, len(v))
		for i := range v {
			xsv[is][i] = make([]float64, nt)
		}
	}

	// rel := make(chan realization, nwrkrs)
	prcd := make(chan bool)
	mth := func() []int {
		o := make([]int, nt)
		for i, t := range frc.T {
			o[i] = int(t.Month()) - 1
		}
		return o
	}()

	go func() {
		for ii, inner := range ev.Outer {
			for _, is := range inner {
				rr := realization{
					i:      is,
					ts:     mth,
					ds:     ev.Sds[is],
					incs:   ev.Incs[is],
					mons:   ev.Mons[is],
					ins:    xsv[is],
					c:      ev.Scids[is],
					ya:     frc.Ya[is],
					ea:     frc.Ea[is],
					deld:   deldsv[ev.Sgw[is]],
					drel:   ev.Drel[is],
					bo:     ev.Bo[is],
					finf:   ev.Finf[is],
					depsto: ev.DepSto[is],
					fcasc:  ev.Fcasc[is],
					fngwc:  ev.Fngwc[ev.Sgw[is]],
					m:      ev.M[ev.Sgw[is]],
					dext:   ev.Dext,
					eafact: ev.Eafact,
					d0:     dm[ev.Sgw[is]],
				}
				if ii == 0 {
					rr.Steady(200., 3) // spin-up
				}
				rel <- rr
				// fmt.Printf("sent %d\n", is)
			}
			<-prcd
		}
	}()

	// r := evalstream(done, rel, nwrkrs)
	var wg sync.WaitGroup
	// var hyd []float64
	// first := true
	for _, inner := range ev.Outer {
		wg.Add(len(inner))
		// if first {
		// 	if len(inner) == 1 {
		// 		signal <- true
		// 		first = false
		// 	}
		// }
		dinner := make([][]float64, ng)
		for i := 0; i < ng; i++ {
			dinner[i] = make([]float64, nt)
		}
		for range inner {
			go func() {
				r := <-res
				is := r.i
				// fmt.Printf("received %d\n", is)

				ig := ev.Sgw[is]

				dsc := ev.Dwnas[is]
				if dsc != nil {
					xsv[dsc[0]][dsc[1]] = r.q // outlet of current sws to downslope sws
				} else {
					hyd = r.q
				}
				for j := 0; j < nt; j++ {
					dinner[ig][j] += r.d[j] // concurrent safe: this operation modifies the members of the slice, not modifying the slice itself
				}

				for i, c := range ev.Scids[is] {
					for j := 0; j < nbins; j++ {
						sae[c] += r.ae[i][j]
						sro[c] += r.ro[i][j]
						srch[c] += r.rch[i][j]
					}
				}

				for _, c := range ev.Mons[is] {
					if a, ok := r.mons[c]; ok {
						writeFloats(outdirprfx+fmt.Sprintf("mon.%d.%d.bin", is, c), a)
					} else {
						panic("wtf")
					}
				}

				wg.Done()
			}()
		}
		wg.Wait()

		// update state
		for ig := range ev.Fngwc {
			for j := 0; j < nt; j++ {
				deldsv[ig][j] = dinner[ig][j]
			}
		}

		prcd <- true
	}

	writeFloats(outdirprfx+"sae.bin", sae)
	writeFloats(outdirprfx+"sro.bin", sro)
	writeFloats(outdirprfx+"srch.bin", srch)
	writeFloats(outdirprfx+"hyd.bin", hyd)
	return hyd
}
