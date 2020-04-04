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
	swscidxr, swsstrmxr map[int][]int       // cross reference sws to cids; sws to stream cell ids
	sws, dsws           map[int]int         // cross reference of cid to sub-watershed ID; map upsws{downsws}
	uca                 map[int]map[int]int // unit contributing areas per sws: swsid{cid{upcnt}}
}

func (r *RTR) subset(topo *tem.TEM, cids, strms []int, outlet int) (*RTR, [][]int, []int) {
	var wg sync.WaitGroup
	var swscidxr map[int][]int  // ordered cids, per sws
	var swsstrmxr map[int][]int // stream cells per sws
	var sws, dsws map[int]int   // sws mapping; downslope watershed mapping
	var sids []int              // slice of subwatershed IDs, safely ordered downslope
	var ord [][]int             // ordered groupings of sws for parallel operations
	var uca map[int]map[int]int // unit contributing areas per sws: swsid{cid{upcnt}}
	if outlet < 0 {
		// log.Fatalf(" RTR.subset error: outlet cell needs to be provided")
		sws = make(map[int]int, len(cids))
		dsws = r.dsws
		sct := make(map[int][]int, len(r.swscidxr))
		for _, cid := range cids {
			if s, ok := r.sws[cid]; ok {
				sws[cid] = s
			} else {
				log.Fatalf(" RTR.subset error: cell not belonging to a sws")
			}
			if _, ok := dsws[sws[cid]]; !ok {
				dsws[sws[cid]] = -1
			}
			if _, ok := sct[sws[cid]]; !ok {
				sct[sws[cid]] = []int{cid}
			} else {
				sct[sws[cid]] = append(sct[sws[cid]], cid)
			}
		}
		sids = mmaths.OrderFromToTree(dsws, -1)
		swscidxr = make(map[int][]int, len(sct))
		for k, v := range sct {
			a := make([]int, len(v)) // note: cids here will be ordered topologically
			copy(a, v)
			swscidxr[k] = a
		}
		uca = r.uca
	} else {
		sws, dsws = make(map[int]int, len(cids)), make(map[int]int, len(r.dsws))
		if len(r.sws) > 0 {
			if _, ok := r.sws[outlet]; !ok {
				log.Fatalf(" RTR.subset error: outlet cell not belonging to a sws")
			}
			sct := make(map[int][]int, len(r.swscidxr))
			osws := r.sws[outlet] // outlet sws (the original sws the outlet cell existed in)
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
						dsws[sws[cid]] = -1 // outlet sws grains to farfield
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
				a := make([]int, len(v)) // note: cids here will be ordered topologically
				copy(a, v)
				swscidxr[k] = a
			}
			uca = make(map[int]map[int]int, len(sct))
			for s := range swscidxr {
				if _, ok := r.uca[s]; ok {
					uca[s] = r.uca[s]
				} else if s == outlet {
					uca[s] = r.uca[osws]
				} else {
					log.Fatalf(" RTR.subset uca error: unknown sws")
				}
			}
			sids = mmaths.OrderFromToTree(dsws, -1)
		} else { // entire model domain is one subwatershed to outlet
			for _, cid := range cids {
				sws[cid] = outlet
			}
			sids = []int{outlet}
			swscidxr = map[int][]int{outlet: cids}
			log.Fatalf(" router.go RTR.subset: to check")
			uca = r.uca
		}
	}

	getSWSord := func() { // build a concurrent-safe ordering of sws
		defer wg.Done()
		// compute sws topology
		tt := mmio.NewTimer()
		defer tt.Print("sws topology build complete")
		// ord = mmaths.OrderedForest(dsws, -1)

		// Top heavy
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
	getSWSstrms := func() {
		defer wg.Done()
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
	}

	wg.Add(2)
	// go getUCA()
	go getSWSord()
	go getSWSstrms()
	wg.Wait()

	return &RTR{
		swscidxr:  swscidxr,
		swsstrmxr: swsstrmxr,
		sws:       sws,
		dsws:      dsws,
		uca:       uca,
	}, ord, sids
}

func (r *RTR) write(dir string) {
	mmio.WriteIMAP(dir+"sws.imap", r.sws)
	if len(r.dsws) > 0 {
		gdsws := make(map[int]int, len(r.sws))
		for k, v := range r.sws {
			if d, ok := r.dsws[v]; ok {
				gdsws[k] = d
			} else {
				fmt.Println("RTR.write error: issue with sws mapping")
			}
		}
		mmio.WriteIMAP(dir+"dsws.imap", gdsws)
	}
}
