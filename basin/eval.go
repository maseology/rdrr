package basin

import (
	"fmt"
	"log"
	"math"
	"sync"
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
	var wg sync.WaitGroup
	dt, y, ep, obs, intvl, nstep := b.getForcings()
	if print {
		if len(b.swsord) == 1 {
			rs := newResults(b, intvl, nstep)
			rs.dt, rs.obs = dt, obs
			var res resulter = &rs
			pp := newSubsample(b, p, Ds, m, -1)
			pp.y, pp.ep, pp.nstep = y, ep, nstep
			pp.eval(Ds, m, res)
			res.report()
		} else {
			transfers := make(map[int][]itran, len(b.rtr.swscidxr))
			nrnds := len(b.swsord)
			for i, k := range b.swsord {
				fmt.Printf("--> round %d (of %d): %d sws\n", i+1, nrnds, len(k))
				chstrans := make(chan stran, len(k))
				for _, sid := range k {
					wg.Add(1)
					dsid := -1
					if d, ok := b.rtr.dsws[sid]; ok {
						transfers[d] = []itran{}
						dsid = d
					}
					go func(sid int) {
						defer wg.Done()
						pp := newSubsample(b, p, Ds, m, sid)
						pp.y, pp.ep, pp.nstep = y, ep, nstep
						if t, ok := transfers[sid]; ok {
							pp.in = make(map[int][]float64, len(t)) // upstream inputs
							for _, v := range t {
								pp.in[pp.xr[v.c]] = v.v
							}
						}
						if sid == b.cid0 { // outlet
							rs := newResults(b, intvl, nstep)
							rs.dt, rs.obs = dt, obs
							var res resulter = &rs
							fmt.Printf(" printing SWS %d\n\n", sid)
							pp.eval(Ds, m, res)
							res.report()
						} else {
							var res resulter = &outflow{}
							fmt.Printf(" running SWS %d\n", sid)
							pp.eval(Ds, m, res)
							chstrans <- stran{s: dsid, i: itran{c: b.ds[sid], v: res.report()}}
						}
					}(sid)
				}
				wg.Wait()
				close(chstrans)
				for t := range chstrans {
					transfers[t.s] = append(transfers[t.s], t.i)
				}
			}
		}
	} else {
		if len(b.swsord) > 0 {
			log.Fatalf(" b.eval todo2\n")
		} else {
			log.Fatalf(" b.eval todo3\n")
		}
	}
	return
}

func (p *subsample) eval(Ds, m float64, res resulter) {
	sim, bf := make([]float64, p.nstep), make([]float64, p.nstep)
	yss, ass, rss, gss, bss := 0., 0., 0., 0., 0.
	// // distributed monitors [mm/yr]
	// gy, ga, gr, gg, gd, gl := make([]float64, b.ncid), make([]float64, b.ncid), make([]float64, b.ncid), make([]float64, b.ncid), make([]float64, b.ncid), make([]float64, b.ncid)

	defer func() { res.getTotals(sim, bf, yss, ass, rss, gss, bss) }()

	dm, s0s := p.dm, p.s0s
	for k := 0; k < p.nstep; k++ {
		ys, ins, as, rs, gs, s1s, bs, dm0 := 0., 0., 0., 0., 0., 0., 0., dm
		for i, v := range p.in {
			p.ws[i].AddToStorage(v[k]) // inflow from up sws
			ins += v[k]
		}
		for i := range p.cids {
			s0 := p.ws[i].Storage()
			a, r, g := p.ws[i].UpdateWT(p.y[k][0], p.ep[k][0], dm+p.drel[i])
			p.ws[i].AddToStorage(r * (1. - p.p0[i]))
			r *= p.p0[i]
			s1 := p.ws[i].Storage()
			s1s += s1

			hruwbal := p.y[k][0] + s0 - (a + r + g + s1)
			if math.Abs(hruwbal) > nearzero {
				fmt.Printf("|hruwbal| = %e\n", hruwbal)
			}

			ys += p.y[k][0]
			as += a
			// gy[i] += y
			// ga[i] += a
			if v, ok := p.strm[i]; ok {
				hb := v * math.Exp((Ds-dm-p.drel[i])/m)
				bs += hb
				r += hb
				// gd[i] += hb
			}
			if p.ds[i] == -1 { // outlet cell
				rs += r
			} else {
				p.ws[p.xr[p.ds[i]]].AddToStorage(r)
			}
			gs += g
			// gr[i] += r
			// gg[i] += g
			// gl[i] += s1
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
			fmt.Printf("(sum)|hruwbal| = %e\n", hruwbal)
		}

		basinwbal := ys + ins + (dm-dm0)*p.fncid + s0s - (as + rs + s1s)
		// basinwbal := (dm - dm0) + (gs-bs)/p.fncid // gwbal
		if math.Abs(basinwbal) > nearzero {
			fmt.Printf("|basinwbal| = %e\n", basinwbal)
		}
		s0s = s1s
	}
	return
}
