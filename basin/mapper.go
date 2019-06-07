package basin

import "github.com/maseology/rdrr/lusg"

// MAPR holds mappings of landuse and surficial geology
type MAPR struct {
	lu       lusg.LandUseColl
	sg       lusg.SurfGeoColl
	ilu, isg map[int]int // cross reference of cid to lu/sg id
}

func (m *MAPR) subset(cids []int) *MAPR {
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

	newMAPR := MAPR{
		lu:  lu,
		sg:  sg,
		ilu: ilu,
		isg: isg,
	}
	return &newMAPR
}
