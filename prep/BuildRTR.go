package prep

import (
	"fmt"
	"log"

	"github.com/maseology/rdrr/model"
)

// BuildRTR returns (and saves) the topological routing scheme amongst sub-basins
func BuildRTR(gobDir string, strc *model.STRC, csws, dsws map[int]int, nsws int) *model.RTR {

	cids, _ := strc.TEM.DownslopeContributingAreaIDs(-1)

	// collect stream cells
	strms, _ := model.BuildStreams(strc, cids)
	sst := make(map[int][]int, nsws)
	for _, c := range strms {
		if s, ok := csws[c]; ok {
			if _, ok := sst[s]; !ok {
				sst[s] = []int{c}
			} else {
				sst[s] = append(sst[s], c)
			}
		}
	}
	swsstrmxr := make(map[int][]int, len(sst))
	for k, v := range sst {
		a := make([]int, len(v))
		copy(a, v)
		swsstrmxr[k] = a
	}

	// compute unit contributing areas
	sct := make(map[int][]int, len(csws))
	for c, s := range csws {
		if _, ok := sct[s]; ok {
			sct[s] = append(sct[s], c)
		} else {
			sct[s] = []int{c}
		}
	}
	swscidxr := make(map[int][]int, len(sct))
	for k, v := range sct {
		a := make([]int, len(v))
		copy(a, v)
		swscidxr[k] = a
	}

	fmt.Print(" building unit contributing areas.. ")
	type col struct {
		s int
		u map[int]int
	}
	ch := make(chan col, len(swscidxr))
	for s, cids := range swscidxr {
		go func(s int, cids []int) {
			m := make(map[int]int, len(cids))
			for _, c := range cids {
				m[c] = 1
				for _, u := range strc.TEM.UpIDs(c) {
					if csws[u] == s { // to be kept within sws
						m[c] += strc.TEM.UnitContributingArea(u)
					}
				}
			}
			ch <- col{s, m}
		}(s, cids)
	}
	uca := make(map[int]map[int]int, len(swscidxr))
	for i := 0; i < len(swscidxr); i++ {
		c := <-ch
		uca[c.s] = c.u
	}
	close(ch)

	rtr := model.RTR{
		SwsCidXR:  swscidxr,  // ordered cids, per sws
		SwsStrmXR: swsstrmxr, // stream cells per sws
		Sws:       csws,      // [cid]sws mapping
		Dsws:      dsws,      // downslope topological watershed routing
		UCA:       uca,       // unit contributing areas per sws: swsid{cid{upcnt}}
	}

	if err := rtr.SaveGob(gobDir + "RTR.gob"); err != nil {
		log.Fatalf(" BuildRTR error: %v", err)
	}

	return &rtr

}
