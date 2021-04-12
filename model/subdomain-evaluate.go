package model

import (
	"fmt"
	"log"
	"math"
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

type version func(p *evaluation, Dinc, m float64, res resulter, monid []int)

// evaluate evaluates a subdomain
func (b *subdomain) evaluate(p *sample, Dinc, m float64, print bool, ver version) (of float64) {

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
		if print { // && b.cid0 > -1 {
			printAggregated(b, p)
		}
		// printTrans(b, transfers, outflw)
	}
	return
}

func printAggregated(b *subdomain, p *sample) {
	if _, ok := mmio.FileExists(fmt.Sprintf("%s%d.wbgt", p.dir, b.cid0)); ok {
		fmt.Println("  aggregating waterbudgets..")
		nstp := len(b.frc.T)
		ty, ta, tg, ts, tds, tb, tdm, tddm, tout, tq := make([]float64, nstp), make([]float64, nstp), make([]float64, nstp), make([]float64, nstp), make([]float64, nstp), make([]float64, nstp), make([]float64, nstp), make([]float64, nstp), make([]float64, nstp), make([]float64, nstp)

		twbal := func(y, i, a, o, b, g, ds, ddm float64, sid, k int) {
			wbal := func(wb float64, sufx string) {
				if math.Abs(wb) > nearzero {
					s1 := fmt.Sprintf("    inputs: atmyld (y=%.5f); inflow (i=%.5f); qbf (b=%.5f); TOTAL = %.5f\n", y, i, b, y+i+b)
					s1 += fmt.Sprintf("   outputs:    aet (e=%.5f); outflw (o=%.5f); gwe (g=%.5f); TOTAL = %.5f\n", a, o, g, a+o+g)
					s1 += fmt.Sprintf("  ins-outs =%10.5f\n", (y+i+b)-(a+o+g))
					s1 += fmt.Sprintf("     gwdef:    ddm = %10.5f   dsto = %10.5f\n", ddm, ds)
					log.Fatalf("printAggregated [%s] waterbalance error |wbal| = %e  (sws %d, step %d)\n%s", sufx, wb, sid, k, s1)
				}
			}
			wbal(y+i+ddm-(a+o+ds), "basin:y+i+ddm-(a+o+ds)")      // basin/sws-wide
			wbal((y+i+b)-(a+o+g+ds), "allHRU:(y+i+b)-(a+o+g+ds)") // sum of all HRUs
			wbal(b-(g+ddm), "GWR:b-(g+ddm)")                      // groundwater res
			// wbal((y)-(a+o), "test")
		}

		dcel := 0.
		for ii, swss := range b.swsord {
			for _, sid := range swss {
				fp := fmt.Sprintf("%s%d.wbgt", p.dir, sid)
				if _, ok := mmio.FileExists(fp); !ok {
					log.Fatalf("printAggregated sws %d waterbudget not printed", sid)
				}
				ncel := float64(len(b.rtr.SwsCidXR[sid]))
				dcel += ncel // weighted average

				if dat, err := mmio.ReadCSV(fp); err != nil { // ys,ins,as,outs,sto,dsto,gs,bs,dm,ddm
					log.Fatalf("printAggregated sws %d waterbudget not printed\n", sid)
				} else {
					for k := 0; k < nstp; k++ {
						twbal(dat[k][0], dat[k][1], dat[k][2], dat[k][3], dat[k][7], dat[k][6], dat[k][5], dat[k][9], sid, k)

						ty[k] += ncel * dat[k][0]
						// ti := ncel * dat[k][1]
						ta[k] += ncel * dat[k][2]
						// to := ncel * dat[k][3]
						tout[k] += ncel * (dat[k][3] - dat[k][1]) // net out = outs - ins
						ts[k] = ncel * dat[k][4]
						tds[k] += ncel * dat[k][5]
						tg[k] += ncel * dat[k][6]
						tb[k] += ncel * dat[k][7]
						tdm[k] = ncel * dat[k][8]
						tddm[k] += ncel * dat[k][9]
					}
					if ii == len(b.swsord)-1 {
						for k := 0; k < nstp; k++ {
							tq[k] += ncel * dat[k][3] // only sum outflows from roots sws (i.e, entire subDomain)
							if math.Abs(tq[k]-tout[k]) > nearzero {
								log.Fatalf("printAggregated outflow summation error\n")
							}
						}
					}
				}
			}
		}

		csvw := mmio.NewCSVwriter(fmt.Sprintf("%saggregated.wbgt.csv", p.dir)) // all-model water budget file
		defer csvw.Close()
		if err := csvw.WriteHead("ys,as,qs,sto,dsto,gs,bs,dm,ddm"); err != nil {
			log.Fatalf("printAggregated %v", err)
		}

		ys, as, qs, bs, gs, dss, dms := 0., 0., 0., 0., 0., 0., 0.
		sum := func(y, a, q, b, g, ds, dm float64) {
			ys += y
			as += a
			qs += q
			bs += b
			gs += g
			dss += ds
			dms += dm
		}
		for i, y := range ty { // weighted average
			sum(y/dcel, ta[i]/dcel, tq[i]/dcel, tb[i]/dcel, tg[i]/dcel, tds[i]/dcel, tddm[i]/dcel)
			twbal(y/dcel, 0., ta[i]/dcel, tq[i]/dcel, tb[i]/dcel, tg[i]/dcel, tds[i]/dcel, tddm[i]/dcel, -1, i)
			csvw.WriteLine(y/dcel, ta[i]/dcel, tq[i]/dcel, ts[i]/dcel, tds[i]/dcel, tg[i]/dcel, tb[i]/dcel, tdm[i]/dcel, tddm[i]/dcel)
		}
		f := 86400 / b.frc.IntervalSec * 365.24 * 1000. / float64(len(ty))
		fmt.Printf("  sums:  y = %.1f  a = %.1f  q = %.1f  g = %.1f  b = %.1f  s = %.1f  dm = %.1f  \n", ys*f, as*f, qs*f, gs*f, bs*f, dss*f, dms*f)
		fmt.Printf("  wbal: (y+b)-(a+q+g+ds) = %.3e\n", (ys+bs-(as+qs+gs+dss))*f)
		fmt.Printf("  gwbl:  b-(g+dm) = %.3e\n", (bs-(gs+dms))*f)
	}
}

// func printTrans(b *subdomain, m map[int][]itran, outflow []float64) {
// 	nstp := 10
// 	txt, _ := mmio.NewTXTwriter("printTrans.txt")
// 	defer txt.Close()
// 	osws := b.rtr.Sws[b.cid0]
// 	xr := make(map[int]int, len(b.rtr.SwsCidXR))
// 	for sws := range b.rtr.SwsCidXR {
// 		xr[b.ds[sws]] = sws
// 	}
// 	for i, k := range b.swsord {
// 		for _, sid := range k {
// 			ss := sid
// 			if sid == osws {
// 				ss = b.cid0
// 			}
// 			txt.Write(fmt.Sprintf("%d ============================================================================== SWS: %d\ninput\t", i+1, ss))
// 			if _, ok := m[ss]; !ok {
// 				txt.Write("none (peak)\n")
// 			} else {
// 				for _, t := range m[ss] {
// 					txt.Write(fmt.Sprintf("%20d", xr[t.c]))
// 				}
// 				txt.Write("\n")
// 				for i := 0; i < nstp; i++ { //} i := range v[0].v {
// 					txt.Write(fmt.Sprintf("\t\t"))
// 					for j := 0; j < len(m[ss]); j++ {
// 						txt.Write(fmt.Sprintf("%20f", m[ss][j].v[i]))
// 					}
// 					txt.Write("\n")
// 				}
// 			}
// 			txt.Write("\n\n")
// 		}
// 	}
// 	txt.Write(fmt.Sprintf("========================== outflow\n"))
// 	for i := 0; i < nstp; i++ {
// 		txt.Write(fmt.Sprintf("\t%20f\n", outflow[i]))
// 	}

// 	// for k, v := range m {
// 	// 	txt.Write(fmt.Sprintf("to SWS: %d\n==========================\ni", k))
// 	// 	for _, t := range v {
// 	// 		txt.Write(fmt.Sprintf(",%d", t.c))
// 	// 	}
// 	// 	txt.Write("\n")
// 	// 	for i := 0; i < 10; i++ { //} i := range v[0].v {
// 	// 		txt.Write(fmt.Sprintf("%d", i))
// 	// 		for j := 0; j < len(v); j++ {
// 	// 			txt.Write(fmt.Sprintf(",%f", v[j].v[i]))
// 	// 		}
// 	// 		txt.Write("\n")
// 	// 	}
// 	// 	txt.Write("\n\n")
// 	// }
// }
