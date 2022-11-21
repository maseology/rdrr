package model

func (dom *Domain) EvaluateQuick(lus []*Surface, dms []float64, xg, xm, gxr []int, prnt bool) []float64 {
	hyd := make([]float64, len(dom.Frc.T)) // output/plotting

	for j := range dom.Frc.T {
		dmg := make([]float64, dom.Ngw)
		ins := make([]float64, dom.Nc)
		for i := range dom.Strc.CIDs { // topologically ordered

			// update land surface
			_, ro, rch := lus[i].Update(dms[xg[i]], ins[i], dom.Frc.Ya[xm[i]][j], dom.Frc.Ea[xm[i]][j])

			// update gw
			dmg[xg[i]] -= rch

			// route flows
			if dom.Strc.DwnXR[i] > -1 {
				ins[dom.Strc.DwnXR[i]] += ro
			} else { // root
				hyd[j] += ro
			}

		}

		// state update: add recharge to gw reservoirs
		for i, g := range dmg {
			dms[i] += g / dom.Mpr.Fngwc[i]
		}

	}
	return hyd // [m/timestep]
}
