package model

func (dom *Domain) Drain(lus []*Surface, dm0 []float64, xg []int, nSteps int) []float64 {
	for j := 0; j < nSteps; j++ {
		ins, dmg := make([]float64, dom.Nc), make([]float64, dom.Ngw)
		for i := range dom.Strc.CIDs {
			_, ro, rch := lus[i].Update(dm0[xg[i]], ins[i], 0., 0.)
			dmg[xg[i]] -= rch
			if dom.Strc.DwnXR[i] > -1 {
				ins[dom.Strc.DwnXR[i]] += ro
			}
		}
		for i, g := range dmg {
			dm0[i] += g / dom.Mpr.Fngwc[i]
		}
	}

	return dm0
}
