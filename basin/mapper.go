package basin

import (
	"github.com/maseology/rdrr/lusg"
)

// MAPR holds mappings of landuse and surficial geology
type MAPR struct {
	lu            lusg.LandUseColl
	sg            lusg.SurfGeoColl
	ilu, isg, sws map[int]int // cross reference of cid to lu/sg id; sub-watershed ID
}

func (m *MAPR) subset(cids []int, outlet int) *MAPR {
	ilu, isg := make(map[int]int, len(cids)), make(map[int]int, len(cids))
	ulutmp, usgtmp := make(map[int]bool, len(m.lu)), make(map[int]bool, len(m.sg))
	for _, cid := range cids {
		l, g := m.ilu[cid], m.isg[cid]
		ilu[cid] = l
		isg[cid] = g
		if _, ok := ulutmp[l]; !ok {
			ulutmp[l] = true
		}
		if _, ok := usgtmp[g]; !ok {
			usgtmp[g] = true
		}
	}

	// collect unique landus and surfgeo types
	lu, sg := make(map[int]lusg.LandUse, len(ulutmp)), make(map[int]lusg.SurfGeo, len(usgtmp))
	for l := range ulutmp {
		lu[l] = m.lu[l]
	}
	for g := range usgtmp {
		sg[g] = m.sg[g]
	}

	sws := make(map[int]int, len(cids))
	if len(m.sws) > 0 {
		osws := m.sws[outlet]
		for _, cid := range cids {
			if i, ok := m.sws[cid]; ok {
				if i == osws {
					sws[cid] = outlet // crops sws to outlet
				} else {
					sws[cid] = i
				}
			} else {
				sws[cid] = cid // main channel outlet cells
			}
		}
	} else { // entire model domain is one subwatershed to outlet
		for _, cid := range cids {
			sws[cid] = outlet
		}
	}

	newMAPR := MAPR{
		lu:  lu,
		sg:  sg,
		sws: sws,
		ilu: ilu,
		isg: isg,
	}
	return &newMAPR
}
