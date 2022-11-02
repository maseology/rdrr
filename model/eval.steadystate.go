package model

import (
	"math"
)

const tolerance = 1.

func (dom *Domain) EvaluateToSteadyState(mmpyr float64, lus []*Surface, cxr map[int]int, xg []int, prnt bool) (dm0 []float64) {

	// lus := make([]Surface, len(luPointers))
	// for i, ll := range luPointers {
	// 	lus[i] = *ll
	// }

	p := mmpyr / 1000. / 365.24 / 4. // [m/6hr] longterm average recharge

	dm0 = make([]float64, dom.Ngw) // initial water deficits (to be solved for)

	gwcells := make([][]int, dom.Ngw)
	for i := range dom.Strc.CIDs {
		gwcells[xg[i]] = append(gwcells[xg[i]], i)
	}

	douts, fuc := make([]bool, dom.Nc), make([]float64, dom.Nc)
	for i, c := range dom.Strc.CIDs {
		fuc[i] = float64(dom.Strc.UpCnt[c])
		d := dom.Strc.DwnXR[i]
		if d == -1 || xg[i] != xg[d] {
			douts[i] = true
		}
	}

	for gi, gcs := range gwcells {

		func(gi int, cids []int) {
			fnc := float64(len(cids))
			dm, ol := 0., 0.
			for j := 0; j < 1e5; j++ {
				o, dmg := 0., 0.
				ins := make([]float64, dom.Nc)
				for _, i := range cids {

					_, ro, rch := lus[i].Update(dm, ins[i]+p, 0.)

					// ro += lus[i].Hru.Sdet.Overflow(0.)

					dmg -= rch
					if douts[i] { // roots and gw res discharge point
						o += ro * fuc[i] / fnc
					} else {
						ins[dom.Strc.DwnXR[i]] += ro
					}
				}
				dm += dmg / fnc
				if math.Abs(ol-o) < tolerance {
					// fmt.Printf("converged on %d\n", j)
					break
				}
				ol = o
			}
		}(gi, gcs)
	}

	return
}
