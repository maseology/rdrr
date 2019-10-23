package basin

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
)

// RTR holds topological info for subwatershed routing
type RTR struct {
	swscidxr, swsstrmxr map[int][]int
	sws, dsws           map[int]int         // cross reference of cid to sub-watershed ID; map upsws{downsws}
	uca                 map[int]map[int]int // unit contributing areas per sws: swsid{cid{upcnt}}
}

func (r *RTR) subset(topo *tem.TEM, cids, strms []int, outlet int) (*RTR, [][]int, []int) {
	var sids []int // slice of subwatershed IDs, safely ordered downslope
	var swscidxr map[int][]int
	var swsstrmxr map[int][]int
	sws, dsws := make(map[int]int, len(cids)), make(map[int]int, len(r.dsws))
	if outlet < 0 {
		log.Fatalf(" RTR.subset error: outlet cell needs to be provided")
	}
	if len(r.sws) > 0 {
		if _, ok := r.sws[outlet]; !ok {
			log.Fatalf(" RTR.subset error: outlet cell not belonging to a sws")
		}
		sct := make(map[int][]int, len(r.swscidxr))
		osws := r.sws[outlet]
		for _, cid := range cids {
			if s, ok := r.sws[cid]; ok {
				if s == osws {
					sws[cid] = outlet // crops sws to outlet
				} else {
					sws[cid] = s
				}
			} else {
				sws[cid] = cid // sacrificial main channel outlet cells
			}
			if _, ok := dsws[sws[cid]]; !ok { // temporarily collect sws's
				if sws[cid] != outlet {
					if r.dsws[sws[cid]] == osws {
						dsws[sws[cid]] = outlet
					} else {
						if _, ok := r.dsws[sws[cid]]; ok {
							dsws[sws[cid]] = r.dsws[sws[cid]]
						} else {
							dsws[sws[cid]] = -1
						}
					}
				} else {
					dsws[sws[cid]] = -1
				}
			}
			if _, ok := sct[sws[cid]]; !ok {
				sct[sws[cid]] = []int{cid}
			} else {
				sct[sws[cid]] = append(sct[sws[cid]], cid)
			}
		}
		swscidxr = make(map[int][]int, len(sct))
		for k, v := range sct {
			a := make([]int, len(v))
			copy(a, v)
			swscidxr[k] = a
		}
		sst := make(map[int][]int, len(r.swsstrmxr))
		for _, c := range strms {
			if s, ok := sws[c]; ok {
				if _, ok := sst[s]; !ok {
					sst[s] = []int{c}
				} else {
					sst[s] = append(sst[s], c)
				}
			}
		}
		swsstrmxr = make(map[int][]int, len(sst))
		for k, v := range sst {
			a := make([]int, len(v))
			copy(a, v)
			swsstrmxr[k] = a
		}
		sids = mmaths.OrderFromToTree(dsws, -1)
	} else { // entire model domain is one subwatershed to outlet
		for _, cid := range cids {
			sws[cid] = outlet
		}
		sids = []int{outlet}
		swscidxr = map[int][]int{outlet: cids}
	}

	var wg sync.WaitGroup
	var ord [][]int
	getOrd := func() {
		defer wg.Done()
		// compute sws topology
		fmt.Println(" building sws topology..")
		// ord = mmaths.OrderedForest(dsws, -1)

		cnt := make(map[int]int, len(sids))
		incr := func(i, v int) {
			if _, ok := cnt[i]; !ok {
				cnt[i] = v + 1
			} else {
				if v+1 > cnt[i] {
					cnt[i] = v + 1
				}
			}
		}
		for _, s := range sids {
			incr(s, 0)
			if v, ok := dsws[s]; ok {
				if v >= 0 { // outlet =-1
					incr(v, cnt[s])
				}
			}
		}
		mord, lord := mmio.InvertMap(cnt)
		ord = make([][]int, len(lord)) // concurrent-safe ordering of subwatersheds
		for i, k := range lord {
			cpy := make([]int, len(mord[k]))
			copy(cpy, mord[k])
			ord[i] = cpy
		}
	}

	var uca map[int]map[int]int
	getUCA := func() {
		defer wg.Done()
		// compute unit contributing areas
		fmt.Println(" building uca..")
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
					for _, u := range topo.UpIDs(c) {
						if sws[u] == s { // to be kept within sws
							m[c] += topo.UnitContributingArea(u)
						}
					}
				}
				ch <- col{s, m}
			}(s, cids)
		}
		// for s, cids := range swscidxr {
		// 	uca[s] = make(map[int]int, len(cids))
		// 	for _, c := range cids {
		// 		uca[s][c] = 1
		// 		for _, u := range topo.UpIDs(c) {
		// 			if sws[u] == s { // to be kept within sws
		// 				uca[s][c] += topo.UnitContributingArea(u)
		// 			}
		// 		}
		// 	}
		// }
		uca = make(map[int]map[int]int, len(swscidxr))
		for i := 0; i < len(swscidxr); i++ {
			c := <-ch
			uca[c.s] = c.u
		}
		close(ch)
	}

	wg.Add(2)
	go getOrd()
	go getUCA()
	wg.Wait()

	return &RTR{
		swscidxr:  swscidxr,
		swsstrmxr: swsstrmxr,
		sws:       sws,
		dsws:      dsws,
		uca:       uca,
	}, ord, sids
}

func (r *RTR) print(dir string) {
	mmio.WriteIMAP(dir+"sws.imap", r.sws)
	if len(r.dsws) > 0 {
		mmio.WriteIMAP(dir+"dsws.imap", r.dsws)
	}
}
