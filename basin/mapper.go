package basin

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/lusg"
)

// MAPR holds mappings of landuse and surficial geology
type MAPR struct {
	LU            lusg.LandUseColl // [luid]LandUseColl
	SG            lusg.SurfGeoColl // [sgid]SurfGeoColl
	Fimp, Fcov    map[int]float64
	LUx, SGx, LKx map[int]int // cross reference of cid to lu/sg/lake id
}

// MaprCXR holds cell-based parameters and indices
type MaprCXR struct {
}

// SaveGob MAPR to gob
func (m *MAPR) SaveGob(fp string) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return fmt.Errorf(" MAPR.SaveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(m); err != nil {
		return fmt.Errorf(" MAPR.SaveGob %v", err)
	}
	return nil
}

// LoadGobMAPR loads
func LoadGobMAPR(fp string) (*MAPR, error) {
	var mapr MAPR
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&mapr)
	if err != nil {
		return nil, err
	}
	return &mapr, nil
}

// func (m *MAPR) write(dir string) error {
// 	mmio.WriteIMAP(dir+"luid.imap", m.LUx)
// 	mmio.WriteIMAP(dir+"sgid.imap", m.SGx)
// 	return nil
// }

func (m *MAPR) writeSubset(dir string, cids []int) error {
	iluss, isgss := make(map[int]int, len(cids)), make(map[int]int, len(cids))
	for _, c := range cids {
		iluss[c] = m.LUx[c]
		isgss[c] = m.SGx[c]
	}
	mmio.WriteIMAP(dir+"luid.imap", iluss)
	mmio.WriteIMAP(dir+"sgid.imap", isgss)
	return nil
}

// // func (m *MAPR) subset(cids []int, outlet int) (*MAPR, []int) {
// // 	// ilu, isg := make(map[int]int, len(cids)), make(map[int]int, len(cids))
// // 	// ulutmp, usgtmp := make(map[int]bool, len(m.lu)), make(map[int]bool, len(m.sg))
// // 	// for _, cid := range cids {
// // 	// 	l, g := m.ilu[cid], m.isg[cid]
// // 	// 	ilu[cid] = l
// // 	// 	isg[cid] = g
// // 	// 	if _, ok := ulutmp[l]; !ok {
// // 	// 		ulutmp[l] = true
// // 	// 	}
// // 	// 	if _, ok := usgtmp[g]; !ok {
// // 	// 		usgtmp[g] = true
// // 	// 	}
// // 	// }

// // 	// // collect unique landus and surfgeo types
// // 	// lu, sg := make(map[int]lusg.LandUse, len(ulutmp)), make(map[int]lusg.SurfGeo, len(usgtmp))
// // 	// for l := range ulutmp {
// // 	// 	lu[l] = m.lu[l]
// // 	// }
// // 	// for g := range usgtmp {
// // 	// 	sg[g] = m.sg[g]
// // 	// }

// // 	sws := make(map[int]int, len(cids))
// // 	var sids []int
// // 	if len(m.sws) > 0 {
// // 		osws := m.sws[outlet]
// // 		for _, cid := range cids {
// // 			if i, ok := m.sws[cid]; ok {
// // 				if i == osws {
// // 					sws[cid] = outlet // crops sws to outlet
// // 				} else {
// // 					sws[cid] = i
// // 				}
// // 			} else {
// // 				sws[cid] = cid // main channel outlet cells
// // 			}
// // 		}
// // 	} else { // entire model domain is one subwatershed to outlet
// // 		for _, cid := range cids {
// // 			sws[cid] = outlet
// // 		}
// // 	}
// // 	// mmio.WriteIMAP("E:/ormgp_rdrr/02HJ007_sws.imap", sws)

// // 	return &MAPR{
// // 		lu:   m.lu,
// // 		sg:   m.sg,
// // 		sws:  sws,
// // 		dsws: m.dsws,
// // 		ilu:  m.ilu,
// // 		isg:  m.isg,
// // 	}, sids
// // }
