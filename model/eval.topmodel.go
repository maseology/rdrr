package model

import (
	"fmt"
	"log"
	"math"
)

func (dom *Domain) TOPMODELonly(lus []*Surface, dms []float64, xg, xm, gxr []int, prnt bool) []float64 {
	hyd := make([]float64, len(dom.Frc.T)) // output/plotting
	dext := 1.                             // extinction depth [m]

	for j := range dom.Frc.T {
		dmg := make([]float64, dom.Ngw)
		ins := make([]float64, dom.Nc)
		for i, c := range dom.Strc.CIDs { // topologically ordered
			s := lus[i]
			d := s.Drel + dms[xg[i]]
			ya, ea := dom.Frc.Ya[xm[i]][j], dom.Frc.Ea[xm[i]][j]
			in0 := ins[i]

			ro, rch, ae := in0+ya, 0., 0.

			if d < 0. { // gw discharge
				f := math.Exp((s.Dinc - d) / s.Tm)
				if math.IsInf(f, 0) {
					panic("TOPMODELonly: inf")
				}
				b := f * s.Bo
				ro += b
				rch -= b + ea
				ae = ea
			} else {
				if d < dext {
					ae = (1. - d/dext) * ea // linear decay
					rch -= ae
				}
				if ro > d {
					ro -= d
					rch += d
				} else {
					rch += ro
					ro = 0.
				}
			}

			dmg[xg[i]] -= rch

			// route flows
			if dom.Strc.DwnXR[i] > -1 {
				ins[dom.Strc.DwnXR[i]] += ro * s.Fcasc
				ins[i] = ro * (1 - s.Fcasc)
			} else { // root
				hyd[j] += ro
			}

			hruwbal := ya + in0 - ae - ro - rch
			if math.Abs(hruwbal) > 1e-5 {
				fmt.Printf("%10d%14.6f%14.6f%14.6f%14.6f%14.6f%14.6f\n", c, hruwbal, in0, ya, ae, ro, rch)
				log.Fatalln("hru wbal error")
			}

		}

		// state update: add recharge to gw reservoirs
		for i, g := range dmg {
			dms[i] += g / dom.Mpr.Fngwc[i]
		}

	}

	return hyd // [m/timestep]
}
