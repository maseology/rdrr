package basin

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

// eval evaluates a subdomain
func (b *subdomain) eval(p *sample, Ds, m float64, print bool) (of float64) {
	if print {
		tt := mmio.NewTimer()
		defer tt.Lap("evaluation completed in")
	}
	var wg sync.WaitGroup
	dt, y, ep, obs, intvl, nstep := b.getForcings()
	if len(b.swsord) == 1 {
		if len(b.rtr.swscidxr) == 1 {
			rs := newResults(b, intvl, nstep)
			rs.dt, rs.obs = dt, obs
			var res resulter = &rs
			pp := newSubsample(b, p, Ds, m, -1, print)
			pp.y, pp.ep, pp.nstep = y, ep, nstep
			pp.eval(Ds, m, res, b.obs[-1])
			of = res.report(print)[0]
		} else {
			log.Fatalf("TODO (subdomain.eval): unordered set of subwatersheds.")
		}
	} else {
		// var outflw []float64
		tt := mmio.NewTimer()
		transfers := make(map[int][]itran, len(b.rtr.swscidxr))
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
					pp := newSubsample(b, p, Ds, m, sid, print)
					pp.y, pp.ep, pp.nstep = y, ep, nstep
					if len(t) > 0 {
						pp.in = make(map[int][]float64, len(t)) // upstream inputs
						for _, v := range t {
							if _, ok := pp.in[pp.xr[v.c]]; ok {
								log.Fatalf("TODO (subdomain.eval): more than one inputs transferred to the same cell: sid: %d, cell: %d\n", sid, v.c)
							}
							pp.in[pp.xr[v.c]] = v.v
						}
					}
					if sid == b.cid0 { // outlet
						rs := newResults(b, intvl, nstep)
						rs.dt, rs.obs = dt, obs
						var res resulter = &rs
						if print {
							fmt.Printf(" printing SWS %d\n\n", sid)
						}
						pp.eval(Ds, m, res, b.obs[sid])
						of = res.report(print)[0]
						// outflw = rs.sim
					} else {
						var res resulter = &outflow{}
						if print {
							fmt.Printf(" running SWS %d\n", sid)
						}
						pp.eval(Ds, m, res, b.obs[sid])
						dsid := -1
						if d, ok := b.rtr.dsws[sid]; ok {
							if _, ok := transfers[d]; !ok {
								transfers[d] = []itran{}
							}
							dsid = d
						}
						chstrans <- stran{s: dsid, i: itran{c: b.ds[sid], v: res.report(false)}}
					}
				}(sid, transfers[sid])
			}
			wg.Wait()
			close(chstrans)
			for t := range chstrans {
				transfers[t.s] = append(transfers[t.s], t.i)
			}
		}
		// printTrans(b, transfers, outflw)
	}
	return
}

func (p *subsample) eval(Ds, m float64, res resulter, monid []int) {
	obs := make(map[int]monitor, len(monid))
	sim, bf := make([]float64, p.nstep), make([]float64, p.nstep)
	yss, ass, rss, gss, bss := 0., 0., 0., 0., 0.
	// distributed monitors [mm/yr]
	ncid := len(p.cids)
	gy, ga, gr, gg, gd, gl := make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid), make([]float64, ncid)

	defer func() {
		res.getTotals(sim, bf, yss, ass, rss, gss, bss)
		for _, v := range obs {
			go v.print()
		}
		g := gmonitor{gy, ga, gr, gg, gd, gl}
		go g.print(p.xr, p.ds, float64(p.nstep))
	}()

	for _, c := range monid {
		obs[p.xr[c]] = monitor{c: c, v: make([]float64, p.nstep)}
	}

	dm, s0s := p.dm, p.s0s
	for k := 0; k < p.nstep; k++ {
		ys, ins, as, rs, gs, s1s, bs, dm0 := 0., 0., 0., 0., 0., 0., 0., dm
		for i, v := range p.in {
			p.ws[i].AddToStorage(v[k]) // inflow from up sws
			ins += v[k]
		}
		for i := range p.cids {
			s0 := p.ws[i].Storage()
			y := p.y[k][0]
			ep := p.ep[k][0]
			drel := p.drel[i]
			p0 := p.p0[i]
			a, r, g := p.ws[i].UpdateWT(y, ep, dm+drel)
			p.ws[i].AddToStorage(r * (1. - p0))
			r *= p0
			s1 := p.ws[i].Storage()
			s1s += s1

			hruwbal := y + s0 - (a + r + g + s1)
			if math.Abs(hruwbal) > nearzero {
				// fmt.Printf("|hruwbal| = %e\n", hruwbal)
				fmt.Print("^")
			}

			ys += y
			as += a
			gy[i] += y
			ga[i] += a
			hb := 0.
			if v, ok := p.strm[i]; ok {
				hb = v * math.Exp((Ds-dm-drel)/m)
				bs += hb
				r += hb
				gd[i] += hb
			}
			if _, ok := obs[i]; ok {
				obs[i].v[k] = r
			}
			if p.ds[i] == -1 { // outlet cell
				rs += r
			} else {
				p.ws[p.xr[p.ds[i]]].AddToStorage(r)
			}
			gs += g
			gr[i] += r
			gg[i] += g
			gl[i] += s1
		}
		yss += ys
		ass += as
		rss += rs
		gss += gs
		bss += bs
		sim[k] = rs
		bf[k] = bs
		dm += (bs - gs) / p.fncid

		hruwbal := ys + ins + bs + s0s - (as + rs + gs + s1s)
		if math.Abs(hruwbal) > nearzero {
			// fmt.Printf("(sum)|hruwbal| = %e\n", hruwbal)
			fmt.Print("*")
		}

		basinwbal := ys + ins + (dm-dm0)*p.fncid + s0s - (as + rs + s1s)
		// basinwbal := (dm - dm0) + (gs-bs)/p.fncid // gwbal
		if math.Abs(basinwbal) > nearzero {
			// fmt.Printf("|basinwbal| = %e\n", basinwbal)
			fmt.Print(".")
		}
		s0s = s1s
	}
	return
}

func printTrans(b *subdomain, m map[int][]itran, outflow []float64) {
	nstp := 10
	txt, _ := mmio.NewTXTwriter("printTrans.txt")
	defer txt.Close()
	osws := b.rtr.sws[b.cid0]
	xr := make(map[int]int, len(b.rtr.swscidxr))
	for sws := range b.rtr.swscidxr {
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
