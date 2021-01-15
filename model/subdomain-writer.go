package model

import (
	"fmt"

	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
)

func (b *subdomain) print() {
	fmt.Println("\nLand Use proportions")
	mLU := make(map[int]int, 10)
	for _, i := range b.cids {
		v := b.mpr.LUx[i]
		if _, ok := mLU[v]; ok {
			mLU[v]++
		} else {
			mLU[v] = 1
		}
	}
	k, v := mmaths.SortMapInt(mLU)
	for i := len(k) - 1; i >= 0; i-- {
		fmt.Printf("%10d %10.1f%%\n", k[i], float64(v[i])*100./float64(len(b.cids)))
	}

	fmt.Println("Surficial Geology proportions")
	mSG := make(map[int]int, 10)
	for _, i := range b.cids {
		v := b.mpr.SGx[i]
		if _, ok := mSG[v]; ok {
			mSG[v]++
		} else {
			mSG[v] = 1
		}
	}
	k, v = mmaths.SortMapInt(mSG)
	for i := len(k) - 1; i >= 0; i-- {
		fmt.Printf("%10d %10.1f%%\n", k[i], float64(v[i])*100./float64(len(b.cids)))
	}
	println()
}

func (b *subdomain) write(dir string) error {
	b.rtr.write(dir + "b.rtr.")
	b.mpr.writeSubset(dir+"b.mpr.", b.cids)
	ucnt, strm := make(map[int]float64, b.ncid), make(map[int]bool, b.nstrm)
	slp := make(map[int]float64, b.ncid)
	mxr := make(map[int]int, b.ncid)
	for _, c := range b.cids {
		ucnt[c] = float64(b.strc.UpCnt[c])
		slp[c] = b.strc.TEM.TEC[c].G
		mxr[c] = b.frc.XR[c]
		if b.strc.UpCnt[c] > 400 {
			strm[c] = true
		}
	}
	mmio.WriteRMAP(dir+"b.strc.t.upcnt.rmap", ucnt, false)
	mmio.WriteRMAP(dir+"b.strc.t.grad.rmap", slp, false)
	mmio.WriteIMAP(dir+"b.frc.mxr.imap", mxr)
	strmca := make(map[int]int, b.ncid)
	for k := range strm {
		strmca[k] = k
		for _, c := range b.strc.TEM.USlp[k] {
			if _, ok := strm[c]; !ok {
				for _, c2 := range b.strc.TEM.ContributingAreaIDs(c) {
					strmca[c2] = k
				}
			}
		}
	}
	mmio.WriteIMAP(dir+"b.strc.t.strmca.imap", strmca)

	// func() { // print summary
	// 	// revxr, _ := mmio.InvertMap(b.frc.XR)
	// 	y, ep := b.frc.D[0], b.frc.D[1]
	// 	nsta := len(y)
	// 	if nsta != len(ep) {
	// 		log.Fatalln(" subdomain.write print summary error 1")
	// 	}
	// 	f := 86400. / b.frc.IntervalSec * 365.24 * 1000. / float64(len(b.frc.T))
	// 	for i := 0; i < nsta; i++ {
	// 		ss, ee := 0., 0.
	// 		for k := range b.frc.T {
	// 			ss += y[i][k]
	// 			ee += ep[i][k]
	// 		}
	// 		fmt.Printf("%d: sy: %.1f  se: %.1f\n", i, ss*f, ee*f) // mm/yr
	// 	}
	// }()

	return nil
}
