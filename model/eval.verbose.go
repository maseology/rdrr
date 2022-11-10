package model

import (
	"fmt"
	"math"

	"github.com/maseology/mmio"
)

func (dom *Domain) EvaluateVerbose(lus []*Surface, dms []float64, xg, xm, gxr []int, prnt bool) []float64 {
	nstp := len(dom.Frc.T)
	fm3s := dom.Strc.Wcell * dom.Strc.Wcell / dom.Frc.IntervalSec                                                                                                 // [m/timestep] to [m³/s]
	hyd := make([]float64, nstp)                                                                                                                                  // output/plotting
	gsya, gaet, gro, grch, gdelsto := make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc) // gridded average outputing
	lns := make([]string, nstp+1)
	// summations
	fnc := float64(dom.Nc)
	sstoL, shyd, sps := 0., 0., 0.
	for _, lu := range lus {
		s := lu.Hru.Storage()
		// gstoL[gxr[i]] = s * 1000. // [mm]
		sstoL += s
	}
	// fmt.Printf("%30s %10s %10s %10s %10s (%6s) %12s\n", "time", "Ya", "aet", "ro", "rch", "delSto", "wbalHRUs")
	for j, t := range dom.Frc.T {
		dmg := make([]float64, dom.Ngw)
		ins := make([]float64, dom.Nc)
		saet, sro, srch, sya, sins, ssto := 0., 0., 0., 0., 0., 0. // summations
		for i := range dom.Strc.CIDs {
			stoL, ya := lus[i].Hru.Storage(), dom.Frc.Ya[xm[i]][j]
			aet, ro, rch := lus[i].Update(dms[xg[i]], ins[i]+ya, dom.Frc.Ea[xm[i]][j])

			dmg[xg[i]] -= rch
			if dom.Strc.DwnXR[i] > -1 {
				ins[dom.Strc.DwnXR[i]] += ro
			} else { // root
				hyd[j] += ro
			}

			// summations
			sto := lus[i].Hru.Storage()
			ssto += sto
			sya += ya
			saet += aet
			sro += ro
			srch += rch
			sins += ins[i]

			//outputs
			gsya[gxr[i]] += ya
			gaet[gxr[i]] += aet
			gro[gxr[i]] += ro - ins[i] // generated runoff
			grch[gxr[i]] += rch
			gdelsto[gxr[i]] += sto - stoL

			// wbal
			hruWbal := ya + ins[i] + stoL - (aet + ro + rch + sto)
			if math.Abs(hruWbal) > nearzero {
				print("o")
			}
		}

		// state update: add recharge to gw reservoirs
		for i, g := range dmg {
			dms[i] += g / dom.Mpr.Fngwc[i]
		}

		// summations
		// shyd += sro / fnc
		shyd += hyd[j] / fnc
		sps += sya / fnc

		// water balances
		allhruWbal := sya + sins + sstoL - (saet + sro + srch + ssto)
		basinWbal := sya/fnc + sstoL/fnc - (saet/fnc + hyd[j]/fnc + srch/fnc + ssto/fnc) // [m]
		if math.Abs(allhruWbal) > nearzero {
			print("*")
		}
		if math.Abs(basinWbal) > nearzero {
			print("-")
		}
		if prnt && j%120 == 0 {
			// fmt.Printf(" %v %10.1f %10.1f %10.1f %10.1f (%6.2f) %12.8f\n", t, sya*1000, saet*1000, sro*1000, srch*1000, (sto-stoL)*1000, allhruwbal)
			fmt.Printf(" %v %10.4f %10.4f %10.4f %10.4f %10.4f %10.4f %10.4f %10.4f (%4.3f) %10.6f %10.6f\n",
				t,
				sya/fnc*1000,
				saet/fnc*1000,
				sro/fnc*1000,
				srch/fnc*1000,
				ssto/fnc*1000,
				dms,
				shyd/float64(j+1)*365.24*4*1000,
				sps/float64(j+1)*365.24*4*1000,
				shyd/sps,
				allhruWbal,
				basinWbal)
		}

		// output/plotting
		if prnt {
			// hyd[j] = sro / fnc
			dmm := func() (o float64) { // mean dm [m]
				for _, v := range dms {
					o += v
				}
				return o / float64(len(dms))
			}()

			lns[j+1] = fmt.Sprintf("%v,%f,%f,%f,%f,%f,%f,%f,%f", t, hyd[j]*fm3s, hyd[j]/fnc*1000, sya/fnc*1000, saet/fnc*1000, srch/fnc*1000, (ssto-sstoL)/fnc*1000, ssto/fnc*1000, dmm) // [mm]
		}

		// reset lasts
		sstoL = ssto

		if j == nstp-1 {
			break
		}
	}

	if prnt {
		// mmplt.ObsSim("hyd.png", dom.Obs.Oq[0], dom.Obs.ToDaily(dom.Frc.T, hyd))
		lns[0] = "date,cms,hyd,Ya,AET,Recharge,DeltaStorage,Storage,MeanDm"
		mmio.DeleteFile("wtrbdgt.csv")
		if err := mmio.WriteStrings("wtrbdgt.csv", lns); err != nil {
			fmt.Println(err)
		}

		// output grids
		f := 4 * 365.24 * 1000 / float64(nstp) // [mm]
		gwbal := make([]float64, dom.Nc)
		for i := range dom.Strc.CIDs {
			gsya[gxr[i]] *= f
			gaet[gxr[i]] *= f
			gro[gxr[i]] *= f
			grch[gxr[i]] *= f
			// gdelsto[gxr[i]] *= f
			gwbal[gxr[i]] = gsya[gxr[i]] - (gaet[gxr[i]] + gro[gxr[i]] + grch[gxr[i]] + gdelsto[gxr[i]])
		}

		writeFloats(dom.Dir+"/output/annual-Ya.bin", gsya)
		writeFloats(dom.Dir+"/output/annual-AET.bin", gaet)
		writeFloats(dom.Dir+"/output/annual-RO.bin", gro)
		writeFloats(dom.Dir+"/output/annual-Rch.bin", grch)
		writeFloats(dom.Dir+"/output/annual-delSto.bin", gdelsto)
		writeFloats(dom.Dir+"/output/final-wbal.bin", gwbal)
	}

	return hyd // [m/timestep]
}
