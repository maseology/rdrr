package model

import (
	"encoding/gob"
	"fmt"
	"os"
)

// MAPR holds mappings of landuse, surficial geology and groundwater zones
//  land use (class): uniform parameters applied to land category
//  surficial geology: ksat/infiltration rate  uniformly applied to surficial geology category
//  ground water zone: paramater assigned to gw zone
type MAPR struct {
	// LU               lusg.LandUseColl // [luid]LandUse
	// GW                     map[int]lusg.TOPMODEL // [gwid]GWzone
	Ksat, Uca, Fimp, Ifct map[int]float64 // [sgid]percolation rate; [cellid]upslope contributing area/cell count; [cellid]fraction impervious; [cellid]interception factor (~=Fcov*LAI)
	LUx, SGx, GWx         map[int]int     // cross reference of cid to lu/sg/gw
	Fngwc                 []float64       // size/area/number of cells for each GWzone
	Strms                 []int           // cell IDs of stream cells
}

// // MaprCXR holds cell-based parameters and indices
// type MaprCXR struct{}

// SaveGob MAPR to gob
func (m *MAPR) SaveGob(fp string) error {
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(" MAPR.SaveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(m); err != nil {
		return fmt.Errorf(" MAPR.SaveGob %v", err)
	}
	f.Close()
	return nil
}

// LoadGobMAPR loads
func LoadGobMAPR(fp string) (*MAPR, error) {
	var mapr MAPR
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&mapr)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &mapr, nil
}

// func (m *MAPR) writeSubset(dir string, cids []int) error {
// 	iluss, isgss := make(map[int]int, len(cids)), make(map[int]int, len(cids))
// 	for _, c := range cids {
// 		iluss[c] = m.LUx[c]
// 		isgss[c] = m.SGx[c]
// 	}
// 	mmio.WriteIMAP(dir+"luid.imap", iluss)
// 	mmio.WriteIMAP(dir+"sgid.imap", isgss)
// 	return nil
// }
