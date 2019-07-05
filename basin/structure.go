package basin

import (
	"github.com/maseology/goHydro/tem"
)

// STRC holds model structural data
type STRC struct {
	t    tem.TEM              // topology
	f    map[int][366]float64 // solar fraction
	a, w float64              // cell area, cell width
}

func (s *STRC) subset(cid0 int) (*STRC, []int, map[int]int) {
	newTEM := s.t.SubSet(cid0)
	cids, ds := newTEM.DownslopeContributingAreaIDs(cid0)
	f := make(map[int][366]float64, len(cids))
	for _, cid := range cids {
		f[cid] = s.f[cid]
	}

	newSTRC := STRC{
		t: newTEM,
		f: f,
		a: s.a,
		w: s.w,
	}
	return &newSTRC, cids, ds
}
