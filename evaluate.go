package rdrr

import (
	"fmt"
	"log"
	"math"

	"github.com/maseology/goHydro/hru"
)

// Evaluate a single run
func (ev *Evaluator) Evaluate(frc *Forcing, nwrkrs int, outdirprfx string) (hyd []float64) {
	return ev.evaluate(frc, nwrkrs, outdirprfx)
}

func (ev *Evaluator) evaluate(frc *Forcing, nwrkrs int, outdirprfx string) []float64 {

	// prep
	nt, ng, ns := len(frc.T), len(ev.Fngwc), len(ev.Scids)
	spr, sae, sro, srch := make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc)
	qout := make([][]float64, ns)
	dmsv := make([]float64, ng)
	x := make([][]hru.Res, ns)
	mon := make([]map[int][]float64, ns)
	for k, cids := range ev.Scids {
		qout[k] = make([]float64, nt)
		x[k] = make([]hru.Res, len(cids))
		for i, d := range ev.DepSto[k] {
			x[k][i].Cap = d
		}
		for _, m := range ev.Mons[k] {
			mon[k][m] = make([]float64, nt)
		}
	}

	for j := range frc.T {
		// fmt.Println(t)

		for k, cids := range ev.Scids {
			ya, ea, dm := frc.Ya[k][j], frc.Ea[k][j], dmsv[ev.Sgw[k]]
			ssae, ssro, ssrch, ssdsto := 0., 0., 0., 0.
			m := ev.M[ev.Sgw[k]]
			for i, c := range cids {

				avail := ea
				dsto0 := x[k][i].Sto
				ro, ae, rch := 0., 0., 0.
				di := ev.Drel[k][i] + dm

				if di < 0. { // gw discharge
					fc := math.Exp(-di / m)
					if math.IsInf(fc, 0) {
						panic("evaluate(): inf")
						fc = 1000.
					}
					b := fc * ev.Bo[k][i]
					ro = x[k][i].Overflow(b + ya)
					rch -= b + avail*ev.Eafact
					ae = avail * ev.Eafact
					avail -= ae
				} else {
					if di < ev.Dext {
						ae = (1. - di/ev.Dext) * avail // linear decay
						rch -= ae
						avail -= ae
					}
					ro = x[k][i].Overflow(ya)
				}

				// Infiltrate surplus/excess mobile water in infiltrated assuming a falling head through a unit length, returns added recharge
				pi := x[k][i].Sto * ev.Finf[k][i]
				x[k][i].Sto -= pi
				rch += pi

				// evaporate from detention storage
				if avail > 0. {
					ae += avail + x[k][i].Overflow(-avail)
				}

				x[k][i].Sto += ro * (1. - ev.Fcasc[k][i])
				ro *= ev.Fcasc[k][i]

				// route flows
				if ids := ev.Sds[k][i]; ids > -1 {
					x[k][ids].Sto += ro
					// ron[ids] += ro
				} else {
					qout[k][j] += ro
				}
				if _, ok := mon[k][c]; ok {
					mon[k][c][j] = ro
				}

				// test for water balance
				hruwbal := ya + dsto0 - x[k][i].Sto - ae - ro - rch
				if math.Abs(hruwbal) > nearzero {
					fmt.Printf("%10d%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, i, hruwbal, x[k][i].Sto, dsto0, ya, ae, ro, rch)
					log.Fatalln("hru wbal error")
				}

				ssae += ae
				ssro += ro
				ssrch += rch
				ssdsto += x[k][i].Sto - dsto0

				spr[c] += ya
				sae[c] += ae
				sro[c] += ro //- (x[k][i].Sto - dsto0) // generated runoff
				srch[c] += rch

				// ron[i] = 0.
			}
			dd := -ssrch / ev.Fngwc[ev.Sgw[k]] // state update: adding recharge decreases the deficit of the gw reservoir
			dmsv[ev.Sgw[k]] += dd              // state update: adding recharge decreases the deficit of the gw reservoir

			// per timestep subwatershed waterbalance
			swswbal := ya - (ssae+ssro+ssrch+ssdsto)/float64(len(cids))
			if math.Abs(swswbal) > nearzero {
				fmt.Printf("%10d%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", k, j, swswbal, ssdsto, ya, ssae, ssro, ssrch)
				log.Fatalln("sws t wbal error")
			}
		}
	}

	hyd := make([]float64, nt)
	for j := range frc.T {
		for k := range ev.Scids {
			hyd[j] += qout[k][j] // / ev.Fnstrm[k]
		}
	}
	lsto := make([]float64, ev.Nc)
	for k, cids := range ev.Scids {
		for i, c := range cids {
			lsto[c] = x[k][i].Sto
		}
	}

	writeFloats(outdirprfx+"spr.bin", spr)
	writeFloats(outdirprfx+"sae.bin", sae)
	writeFloats(outdirprfx+"sro.bin", sro)
	writeFloats(outdirprfx+"srch.bin", srch)
	writeFloats(outdirprfx+"lsto.bin", lsto)
	// writeFloats(outdirprfx+"hyd.bin", hyd)

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
