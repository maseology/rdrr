package model

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/mmio"
)

type itran struct {
	v []float64
	c int
}
type stran struct {
	i itran
	s int
}

// evaluate evaluates a subdomain
func (b *subdomain) evaluate(p *sample, Dinc, m float64, print bool) (of float64) {

	ver := evalMC

	nstep := len(b.frc.T)

	if print {
		tt := mmio.NewTimer()
		defer tt.Lap("\nevaluation completed in")
	}
	var wg sync.WaitGroup
	// dt, y, ep, obs, intvl, nstep := b.getForcings()
	if len(b.swsord) == 1 {
		if len(b.rtr.SwsCidXR) == 1 {
			rs := newResults(b, nstep)
			rs.dt, rs.obs = b.frc.T, b.frc.O[0]
			var res resulter = &rs
			pp := newEvaluation(b, p, Dinc, m, b.cid0, print)
			ver(&pp, Dinc, m, res, b.mon[b.cid0])
			of = res.report(print)[0]
		} else {
			log.Fatalf("TODO (subdomain.eval): unordered set of subwatersheds.")
		}
	} else {
		// var outflw []float64
		tt := mmio.NewTimer()
		transfers := make(map[int][]itran, len(b.rtr.SwsCidXR))
		nrnds := len(b.swsord)
		for i, k := range b.swsord {
			if print {
				// fmt.Printf("--> round %d (of %d): %d sws\n", i+1, nrnds, len(k))
				tt.Print(fmt.Sprintf("--> round %d (of %d): %d sws", i+1, nrnds, len(k)))
			}
			chstrans := make(chan stran, len(k))
			for _, sid := range k {
				wg.Add(1)
				go func(sid int, t []itran) {
					defer wg.Done()
					pp := newEvaluation(b, p, Dinc, m, sid, print)
					if len(t) > 0 {
						pp.sources = make(map[int][]float64, len(t)) // upstream inputs
						for _, v := range t {
							if _, ok := pp.sources[pp.cxr[v.c]]; ok {
								for i, vv := range v.v {
									pp.sources[pp.cxr[v.c]][i] += vv
								}
								// log.Fatalf("TODO (subdomain.eval): more than one sources transferred to the same cell: sid: %d, cell: %d\n", sid, v.c)
							} else {
								pp.sources[pp.cxr[v.c]] = v.v
							}
						}
					}
					if sid == b.cid0 { // outlet
						rs := newResults(b, nstep)
						rs.dt = b.frc.T
						if b.frc.O != nil {
							rs.obs = b.frc.O[0] //* pp.intvl / pp.ca
						}
						var res resulter = &rs
						if print {
							fmt.Printf(" printing SWS %d\n\n", sid)
						}
						ver(&pp, Dinc, m, res, b.mon[sid])
						of = res.report(print)[0]
						// outflw = rs.sim
					} else {
						var res resulter = &outflow{}
						if print {
							fmt.Printf(" running SWS %d\n", sid)
						}
						ver(&pp, Dinc, m, res, b.mon[sid])
						dsid := -1
						if d, ok := b.rtr.Dsws[sid]; ok {
							// if _, ok := transfers[d]; !ok {
							// 	transfers[d] = []itran{} //// concurrent map write potential (now fixed below)
							// }
							dsid = d
						}
						chstrans <- stran{s: dsid, i: itran{c: b.ds[sid], v: res.report(false)}}
					}
				}(sid, transfers[sid])
			}
			wg.Wait()
			close(chstrans)
			for t := range chstrans {
				if _, ok := transfers[t.s]; !ok {
					transfers[t.s] = []itran{t.i}
				} else {
					transfers[t.s] = append(transfers[t.s], t.i)
				}
			}
		}
		// printTrans(b, transfers, outflw)
	}
	return
}

func printTrans(b *subdomain, m map[int][]itran, outflow []float64) {
	nstp := 10
	txt, _ := mmio.NewTXTwriter("printTrans.txt")
	defer txt.Close()
	osws := b.rtr.Sws[b.cid0]
	xr := make(map[int]int, len(b.rtr.SwsCidXR))
	for sws := range b.rtr.SwsCidXR {
		xr[b.ds[sws]] = sws
	}
	for i, k := range b.swsord {
		for _, sid := range k {
			ss := sid
			if sid == osws {
				ss = b.cid0
			}
			txt.Write(fmt.Sprintf("%d ============================================================================== SWS: %d\ninput\t", i+1, ss))
			if _, ok := m[ss]; !ok {
				txt.Write("none (peak)\n")
			} else {
				for _, t := range m[ss] {
					txt.Write(fmt.Sprintf("%20d", xr[t.c]))
				}
				txt.Write("\n")
				for i := 0; i < nstp; i++ { //} i := range v[0].v {
					txt.Write(fmt.Sprintf("\t\t"))
					for j := 0; j < len(m[ss]); j++ {
						txt.Write(fmt.Sprintf("%20f", m[ss][j].v[i]))
					}
					txt.Write("\n")
				}
			}
			txt.Write("\n\n")
		}
	}
	txt.Write(fmt.Sprintf("========================== outflow\n"))
	for i := 0; i < nstp; i++ {
		txt.Write(fmt.Sprintf("\t%20f\n", outflow[i]))
	}

	// for k, v := range m {
	// 	txt.Write(fmt.Sprintf("to SWS: %d\n==========================\ni", k))
	// 	for _, t := range v {
	// 		txt.Write(fmt.Sprintf(",%d", t.c))
	// 	}
	// 	txt.Write("\n")
	// 	for i := 0; i < 10; i++ { //} i := range v[0].v {
	// 		txt.Write(fmt.Sprintf("%d", i))
	// 		for j := 0; j < len(v); j++ {
	// 			txt.Write(fmt.Sprintf(",%f", v[j].v[i]))
	// 		}
	// 		txt.Write("\n")
	// 	}
	// 	txt.Write("\n\n")
	// }
}
