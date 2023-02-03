package rdrr

import (
	"log"
	"sort"
	"time"

	"github.com/maseology/mmio"
)

func dayDate(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local).Unix()
}

func (s *Structure) CollectObservations(frc *Forcing, obsfp string) *Observations {
	var mons []int
	if _, ok := mmio.FileExists(obsfp); ok {
		var err error
		if mons, err = mmio.ReadInts(obsfp); err != nil {
			log.Fatalf("%v", err)
		}
	}

	mt, cmt, dicT := make([]int, len(frc.T)), make([]float64, 12), make(map[int64][]int)
	for k, t := range frc.T {
		mt[k] = int(t.Month())
		cmt[mt[k]-1]++
		dicT[dayDate(t)] = append(dicT[dayDate(t)], k)
	}
	lstT, td, tx := make([]int64, 0, len(dicT)), make([]time.Time, len(dicT)), make([]int, len(frc.T))
	for u := range dicT {
		lstT = append(lstT, u)
	}
	sort.Slice(lstT, func(i, j int) bool { return lstT[i] < lstT[j] })
	for i, ud := range lstT {
		td[i] = time.Unix(ud, 0)
		if a, ok := dicT[ud]; ok {
			for _, ii := range a {
				tx[ii] = i
			}
		} else {
			panic("CollectObservations error")
		}
	}

	obs := &Observations{
		Td:   td,
		Oq:   make([][]float64, 0),
		Oqxr: make([]int, 0),
		txr:  tx,
		// mt:       mt,
		// cmt:      cmt,
		Mons:     mons,
		cellarea: s.GD.Cwidth * s.GD.Cwidth,
		// dayfraction: frc.IntervalSec / 84600.,
	}

	m := make(map[int]int)
	for i, c := range s.Cids {
		m[c] = i
	}
	rootdir := mmio.GetFileDir(obsfp)
	if mmio.DirExists(rootdir + "/obs/") {
		obs.AddFluxCsv(rootdir+"/obs/", m)
	}

	return obs
}
