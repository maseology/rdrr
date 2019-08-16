package basin

import (
	"log"

	"github.com/maseology/mmaths"
)

// RTR holds topological info for subwatershed routing
type RTR struct {
	sws, dsws map[int]int // cross reference of cid to sub-watershed ID; map upsws{downsws}
}

func (r *RTR) subset(cids []int, outlet int) (*RTR, []int) {
	var sids []int
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

	return &RTR{
		sws:  sws,
		dsws: r.dsws,
	}, sids
}
