package model

import "fmt"

func (dom *Domain) EvaluateMC(lus []*Surface, dms []float64, xg, xm, gxr []int, mcrun int) []float64 {
	hyd := make([]float64, len(dom.Frc.T))                                                                              // output/plotting
	gsya, gaet, gro, grch := make([][]float64, 12), make([][]float64, 12), make([][]float64, 12), make([][]float64, 12) // gridded average outputing
	for i := 0; i < 12; i++ {
		gsya[i] = make([]float64, dom.Nc)
		gaet[i] = make([]float64, dom.Nc)
		gro[i] = make([]float64, dom.Nc)
		grch[i] = make([]float64, dom.Nc)
	}
	for j := range dom.Frc.T {
		dmg := make([]float64, dom.Ngw)
		ins := make([]float64, dom.Nc)
		for i := range dom.Strc.CIDs { // topologically ordered

			// update land surface
			aet, ro, rch := lus[i].Update(dms[xg[i]], ins[i], dom.Frc.Ya[xm[i]][j], dom.Frc.Ea[xm[i]][j])

			// update gw
			dmg[xg[i]] -= rch

			// route flows
			if dom.Strc.DwnXR[i] > -1 {
				ins[dom.Strc.DwnXR[i]] += ro
			} else { // root
				hyd[j] += ro
			}

			mx := dom.Obs.mt[j] - 1
			gsya[mx][gxr[i]] += dom.Frc.Ya[xm[i]][j]
			gaet[mx][gxr[i]] += aet
			gro[mx][gxr[i]] += ro - ins[i] // generated runoff
			grch[mx][gxr[i]] += rch

		}

		// state update: add recharge to gw reservoirs
		for i, g := range dmg {
			dms[i] += g / dom.Mpr.Fngwc[i]
		}
	}

	for j := 0; j < 12; j++ {
		f := 4 * 30 * 1000 / dom.Obs.cmt[j] // [mm/mo]
		for i := range dom.Strc.CIDs {
			gsya[j][gxr[i]] *= f
			gaet[j][gxr[i]] *= f
			gro[j][gxr[i]] *= f
			grch[j][gxr[i]] *= f
		}
	}

	write2Floats(fmt.Sprintf("%s/output/monthly-Ya-%d.bin", dom.Dir, mcrun), gsya)
	write2Floats(fmt.Sprintf("%s/output/monthly-AET-%d.bin", dom.Dir, mcrun), gaet)
	write2Floats(fmt.Sprintf("%s/output/monthly-RO-%d.bin", dom.Dir, mcrun), gro)
	write2Floats(fmt.Sprintf("%s/output/monthly-Rch-%d.bin", dom.Dir, mcrun), grch)

	return hyd // [m/timestep]
}
