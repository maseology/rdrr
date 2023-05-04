package rdrr

import (
	"fmt"

	"github.com/maseology/mmio"
)

func (ev *Evaluator) EvaluateNaive(frc *Forcing, nc, dummy int, outdir string) (hyd []float64) {
	nt, ng, ns := len(frc.T), len(ev.Fngwc), len(ev.Scids)
	hyds := make([][]float64, ns)
	spr, sae, sro, srch, lsto := make([]float64, nc), make([]float64, nc), make([]float64, nc), make([]float64, nc), make([]float64, nc)
	xsv, deldsv, mth := ev.prep(frc.T)
	dm := make([]float64, ng)
	nout := 0

	for _, inner := range ev.Outer {
		for _, is := range inner {
			println(is)
			rel := realization{
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
				// dext:   ev.Dext,
				eafact: ev.Eafact,
				d0:     dm[ev.Sgw[is]],
			}
			// if ii == 0 {
			// 	rel.Steady(200., 3) // spin-up
			// }

			res := rel.rdrr()

			dm[ev.Sgw[is]] = res.dmlast // setting last d of last round to initial d, this should help spinup issues
			dsc := ev.Dwnas[is]
			if dsc != nil {
				xsv[dsc[0]][dsc[1]] = res.q // outlet of current sws to downslope sws
			} else {
				hyd = res.q
				nout++
			}
			hyds[is] = res.q

			ig := ev.Sgw[is]
			for j := 0; j < nt; j++ {
				deldsv[ig][j] = res.d[j] // / ev.Fngwc[ig]
			}

			for i, c := range ev.Scids[is] {
				for j := 0; j < nbins; j++ {
					spr[c] += res.pr[i][j]
					sae[c] += res.ae[i][j]
					sro[c] += res.ro[i][j]
					srch[c] += res.rch[i][j]
				}
				lsto[c] = res.s[i]
			}

			for _, c := range ev.Mons[is] {
				if a, ok := res.mons[c]; ok {
					writeFloats(outdir+fmt.Sprintf("mon.%d.%d.bin", is, c), a)
				} else {
					panic("wtf")
				}
			}
		}
	}
	if nout != 1 {
		println(nout)
		print("TODO: multiple model outlets")
	}

	mmio.WriteCsvDateFloats("hydALL.csv", "date,i0,i1,i2,i3,i4,i5,i6,i7,i8,i9,i10,i11,i12,i13,i14,i15,i16,i17,i18,i19,i20", frc.T, hyds...)
	// mmio.WriteCsvDateFloats("hyd10.csv", "date,i10", frc.T, hyds[10])

	writeFloats(outdir+"spr.bin", spr)
	writeFloats(outdir+"sae.bin", sae)
	writeFloats(outdir+"sro.bin", sro)
	writeFloats(outdir+"srch.bin", srch)
	writeFloats(outdir+"lsto.bin", lsto)

	return hyd // [m/timestep]
}
