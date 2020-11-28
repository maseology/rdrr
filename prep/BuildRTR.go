package prep

import (
	"fmt"
	"log"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

// BuildRTR returns (and saves) the topological routing scheme amongst sub-basins
func BuildRTR(gobDir, topoFP string, strc *model.STRC, swsc map[int][]int, csws map[int]int) *model.RTR {

	cids, _ := strc.TEM.DownslopeContributingAreaIDs(-1)

	// collect topology
	var dsws map[int]int
	nsws := len(swsc)
	if _, ok := mmio.FileExists(topoFP); ok {
		d, err := mmio.ReadCSV(topoFP)
		if err != nil {
			log.Fatalf(" Loader.readSWS: error reading %s: %v\n", topoFP, err)
		}
		// dsws = make(map[int]int, len(d)) // note: swsids not contained within dsws drain to farfield
		// for _, ln := range d {
		// 	dsws[int(ln[1])] = int(ln[2]) // linkID,upstream_swsID,downstream_swsID
		// }
		dsws = make(map[int]int, nsws) // note: swsids not contained within dsws drain to farfield
		for _, ln := range d {
			if _, ok := swsc[int(ln[1])]; ok {
				if _, ok := swsc[int(ln[2])]; ok {
					dsws[int(ln[1])] = int(ln[2]) // linkID,upstream_swsID,downstream_swsID
				}
			}
		}
	} else {
		// fmt.Printf(" warning: sws topology (*.topo) not found\n")
		log.Fatalf(" BuildRTR error: sws topology (*.topo) not found: %s", topoFP)
	}

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
