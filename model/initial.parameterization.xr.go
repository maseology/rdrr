package model

import (
	"github.com/maseology/mmio"
)

func (dom *Domain) GetCrossreferences(prnt bool) ([]int, []int, map[int]int) {
	tt := mmio.NewTimer()

	xg, xm := make([]int, dom.Nc), make([]int, dom.Nc) // cross-referencing
	m := make(map[int]int, dom.Nc+1)                   // cross-referencing cell id to array index
	m[-1] = -1

	for i, c := range dom.Strc.CIDs {
		// cross-referencing
		m[c] = i
		xg[i] = func() int {
			if gid, ok := dom.Mpr.GWx[c]; ok {
				return gid
			}
			panic("gw xr build error")
		}()
		xm[i] = func() int {
			if mid, ok := dom.Frc.XR[c]; ok {
				return mid
			}
			panic("met xr build error")
		}()
	}

	if prnt {
		tt.Print("cross-reference build complete")
	}

	return xg, xm, m
}
