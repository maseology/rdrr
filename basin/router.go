package basin

import (
	"log"

	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
)

// RTR holds topological info for subwatershed routing
type RTR struct {
	sws, dsws map[int]int // cross reference of cid to sub-watershed ID; map upsws{downsws}
}

func (r *RTR) subset(cids []int, outlet int) (*RTR, [][]int, []int) {
	var sids []int // slice of subwatershed IDs, safely ordered downslope
	sws, dsws := make(map[int]int, len(cids)), make(map[int]int, len(r.dsws))
	if len(r.sws) > 0 {
		osws := r.sws[outlet]
		for _, cid := range cids {
			if i, ok := r.sws[cid]; ok {
				if i == osws {
					sws[cid] = outlet // crops sws to outlet
				} else {
					sws[cid] = i
				}
			} else {
				sws[cid] = cid // main channel outlet cells
			}
			if _, ok := dsws[sws[cid]]; !ok { // temporarily collect sws's
				if sws[cid] != outlet {
					if r.dsws[sws[cid]] == osws {
						dsws[sws[cid]] = outlet
					} else {
						dsws[sws[cid]] = r.dsws[sws[cid]]
					}
				} else {
					dsws[sws[cid]] = -1
				}
			}
		}
		sids = mmaths.OrderFromToTree(dsws, -1)
	} else { // entire model domain is one subwatershed to outlet
		log.Fatalf(" RTR.subset: to check...")
		for _, cid := range cids {
			sws[cid] = outlet
		}
		sids = []int{outlet}
	}

	cnt := make(map[int]int, len(sids))
	for _, s := range sids {
		incr := func(i, v int) {
			if _, ok := cnt[i]; !ok {
				cnt[i] = v + 1
			} else {
				if v+1 > cnt[i] {
					cnt[i] = v + 1
				}
			}
		}
		incr(s, 0)
		if v, ok := dsws[s]; ok {
			if v >= 0 { // outlet =-1
				incr(v, cnt[s])
			}
		}
	}
	mord, lord := mmio.InvertMap(cnt)
	ord := make([][]int, len(lord)) // concurrent-safe ordering of subwatersheds
	for i, k := range lord {
		cpy := make([]int, len(mord[k]))
		copy(cpy, mord[k])
		ord[i] = cpy
	}

	return &RTR{
		sws:  sws,
		dsws: r.dsws,
	}, ord, sids
}
