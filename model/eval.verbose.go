package model

import (
	"fmt"
	"math"

	"github.com/maseology/mmio"
)

func (dom *Domain) EvaluateVerbose(lus []*Surface, dms []float64, xg, xm, gxr []int, prnt bool) []float64 {
	nstp := len(dom.Frc.T)
	fm3s := dom.Strc.Wcell * dom.Strc.Wcell / dom.Frc.IntervalSec                                                                                              // [m/timestep] to [m³/s]
	hyd := make([]float64, nstp)                                                                                                                               // output/plotting
	gsya, gaet, gro, grch, gsto := make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc) // gridded average outputing
	lns := make([]string, nstp+1)
	// summations
	fnc := float64(dom.Nc)
	stoL, shyd, sps := 0., 0., 0.
	for _, lu := range lus {
		stoL += lu.Hru.Storage()
	}
	// fmt.Printf("%30s %10s %10s %10s %10s (%6s) %12s\n", "time", "Ya", "aet", "ro", "rch", "delSto", "wbalHRUs")
	for j, t := range dom.Frc.T {
		dmg := make([]float64, dom.Ngw)
		ins := make([]float64, dom.Nc)
		saet, sro, srch, sya, sins, sto := 0., 0., 0., 0., 0., 0. // summations
		for i := range dom.Strc.CIDs {

			aet, ro, rch := lus[i].Update(dms[xg[i]], ins[i]+dom.Frc.Ya[xm[i]][j], dom.Frc.Ea[xm[i]][j])

			dmg[xg[i]] -= rch
			if dom.Strc.DwnXR[i] > -1 {
				ins[dom.Strc.DwnXR[i]] += ro
			} else { // root
				hyd[j] += ro
			}

			// summations
			sto += lus[i].Hru.Storage()
			sya += dom.Frc.Ya[xm[i]][j]
			saet += aet
			sro += ro
			srch += rch
			sins += ins[i]

			//outputs
			gsya[gxr[i]] += dom.Frc.Ya[xm[i]][j]
			gaet[gxr[i]] += aet
			gro[gxr[i]] += ro
			grch[gxr[i]] += rch
			// gsto[gxr[i]] += sto
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
		allhruWbal := sya + sins + stoL - (saet + sro + srch + sto)
		basinWbal := sya/fnc + stoL/fnc - (saet/fnc + hyd[j]/fnc + srch/fnc + sto/fnc) // [m]
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
				sto/fnc*1000,
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

			lns[j+1] = fmt.Sprintf("%v,%f,%f,%f,%f,%f,%f,%f,%f", t, hyd[j]*fm3s, hyd[j]/fnc*1000, sya/fnc*1000, saet/fnc*1000, srch/fnc*1000, (sto-stoL)/fnc*1000, sto/fnc*1000, dmm) // [mm]
		}

		// reset lasts
		stoL = sto

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
		f := 4 * 365.24 * 1000 / float64(nstp) // [mm]
		for i := range dom.Strc.CIDs {
			gsya[gxr[i]] *= f
			gaet[gxr[i]] *= f
			gro[gxr[i]] *= f
			grch[gxr[i]] *= f
			gsto[gxr[i]] = lus[i].Hru.Storage() * 1000. // [mm]
		}

		writeFloats(dom.Dir+"/output/annual-Ya.bin", gsya)
		writeFloats(dom.Dir+"/output/annual-AET.bin", gaet)
		writeFloats(dom.Dir+"/output/annual-RO.bin", gro)
		writeFloats(dom.Dir+"/output/annual-Rch.bin", grch)
		writeFloats(dom.Dir+"/output/final-Storage.bin", gsto)
	} // output grids

	return hyd // [m/timestep]
}
