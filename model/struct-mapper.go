package model

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/lusg"
)

// MAPR holds mappings of landuse and surficial geology
type MAPR struct {
	LU lusg.LandUseColl // [luid]LandUseColl
	// SG         lusg.SurfGeoColl // [sgid]SurfGeoColl
	Ksat, Fimp, Ifct map[int]float64 // fraction impervious; interception factor, total interception=Ifct*Fcov*IntSto
	LUx, SGx         map[int]int     // cross reference of cid to lu/sg
}

// MaprCXR holds cell-based parameters and indices
type MaprCXR struct{}

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
