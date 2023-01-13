package model

import (
	"encoding/gob"
	"fmt"
	"os"
)

// STRC holds model structural data
type STRC struct {
	UpSlps      map[int][]int   // slice of upslope cells
	DwnGrad     map[int]float64 // gradient (slope) of cell
	UpCnt       map[int]int     // cell upslope count (unit contributing area)
	CIDs, DwnXR []int           // topologically-ordered (grid)cell IDs; downslope cell array index
	// Acell, Wcell float64         // cell area, cell width
	Wcell float64 // cell width
	CID0  int     // cell id of outlet cell.  <0 for all cells
}

// SaveGob STRC to gob
func (s *STRC) SaveGob(fp string) error {
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(" STRC.Save %v", err)
	}
	if err := gob.NewEncoder(f).Encode(s); err != nil {
		return fmt.Errorf(" STRC.Save %v", err)
	}
	f.Close()
	return nil
}

// LoadGobSTRC loads
func LoadGobSTRC(fp string) (*STRC, error) {
	var strc STRC
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&strc)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &strc, nil
}
