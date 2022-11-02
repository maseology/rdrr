package model

import (
	"fmt"

	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
)

func (d *Domain) Print() {
	fmt.Println()
	fncell := float64(len(d.Strc.UpCnt))
	fmt.Printf("  catchment area: %.1f km² (%s cells);\n", fncell*d.Strc.Wcell*d.Strc.Wcell/1000./1000., mmio.Thousands(int64(fncell)))
	fmt.Printf("  model period %v to %v;\n  nsteps = %d;\n  interval %.0f sec;\n", d.Frc.T[0], d.Frc.T[len(d.Frc.T)-1], len(d.Frc.T), d.Frc.IntervalSec)
	fmt.Printf("  n monitors = %d; n observations = %d;\n", len(d.Obs.mons), len(d.Obs.Oqxr))

	mLU := make(map[int]int)
	for c := range d.Strc.UpCnt {
		v := d.Mpr.LUx[c]
		mLU[v]++
	}
	fmt.Printf("\n Land Use proportions (%d)\n", len(mLU))
	k, v := mmaths.SortMapInt(mLU)
	for i := len(k) - 1; i >= 0; i-- {
		fmt.Printf("%10d %10.1f%%\n", k[i], float64(v[i])*100./fncell)
	}

	mSG := make(map[int]int)
	for c := range d.Strc.UpCnt {
		v := d.Mpr.SGx[c]
		mSG[v]++
	}
	fmt.Printf(" Surficial Geology proportions (%d)\n", len(mSG))
	k, v = mmaths.SortMapInt(mSG)
	for i := len(k) - 1; i >= 0; i-- {
		fmt.Printf("%10d %10.1f%%\n", k[i], float64(v[i])*100./fncell)
	}

	mGW := make(map[int]int)
	for c := range d.Strc.UpCnt {
		v := d.Mpr.GWx[c]
		mGW[v]++
	}
	fmt.Printf(" Groundwater region proportions (%d)\n", len(mGW))
	k, v = mmaths.SortMapInt(mGW)
	for i := len(k) - 1; i >= 0; i-- {
		fmt.Printf("%10d %10.1f%%\n", k[i], float64(v[i])*100./fncell)
	}
	fmt.Printf("\n")
}
