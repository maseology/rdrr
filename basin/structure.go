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

// func (s *STRC) subset(gd *grid.Definition, sif map[int][366]float64, cid0 int) (*STRC, []int, map[int]int) {
// 	newTEM := s.t.SubSet(cid0)
// 	cids, ds := newTEM.DownslopeContributingAreaIDs(cid0)
// 	// f := make(map[int][366]float64, len(cids))

// 	// type kv struct {
// 	// 	k int
// 	// 	v [366]float64
// 	// }
// 	// var wg1 sync.WaitGroup
// 	// ch := make(chan kv, len(cids))
// 	// psi := func(tec tem.TEC, x, y float64, cid int) {
// 	// 	defer wg1.Done()
// 	// 	latitude, _, err := UTM.ToLatLon(x, y, 17, "", true)
// 	// 	if err != nil {
// 	// 		log.Fatalf(" STRC.subset (SolIrradFrac) error: %v -- (x,y)=(%f, %f); cid: %d\n", err, x, y, cid)
// 	// 	}
// 	// 	si := solirrad.New(latitude, math.Tan(tec.S), math.Pi/2.-tec.A)
// 	// 	ch <- kv{k: cid, v: si.PSIfactor()}
// 	// }

// 	// for _, cid := range cids {
// 	// 	if v, ok := s.f[cid]; ok {
// 	// 		f[cid] = v
// 	// 	} else {
// 	// 		wg1.Add(1)
// 	// 		c := gd.Coord[cid]
// 	// 		go psi(newTEM.TEC[cid], c.X, c.Y, cid)
// 	// 	}
// 	// }
// 	// wg1.Wait()
// 	// close(ch)
// 	// for kv := range ch {
// 	// 	f[kv.k] = kv.v
// 	// }

// 	newSTRC := STRC{
// 		t: newTEM,
// 		f: sif,
// 		a: s.a,
// 		w: s.w,
// 	}
// 	return &newSTRC, cids, ds
// }
