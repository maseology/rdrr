package rdrr

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/goHydro/grid"
)

type Structure struct {
	GD              *grid.Definition
	Cids, Ds, Upcnt []int     // topologically safe order of grid-cell IDs that make up the model domain; down-slope cell array index; array of model outlet/farfield cells
	Dwngrad         []float64 // downslope gradient (beta)
	Nc              int       // number of cells, groundwater zones
}

func (s *Structure) Checkandprint(chkdirprfx string) {

	// output
	mx := make(map[int]int, s.Nc)
	for i, c := range s.Cids {
		mx[c] = i
	}

	aids, cids, ds, upcnt := s.GD.NullInt32(-9999), s.GD.NullInt32(-9999), s.GD.NullInt32(-9999), s.GD.NullInt32(-9999)
	dwngrad := s.GD.NullArray(-9999.)
	for _, c := range s.GD.Sactives {
		if i, ok := mx[c]; ok {
			aids[c] = int32(i)
			cids[c] = int32(s.Cids[i])
			ds[c] = int32(s.Ds[i])
			upcnt[c] = int32(s.Upcnt[i])
			dwngrad[c] = s.Dwngrad[i]
		}
	}

	writeInts(chkdirprfx+"structure.aids.indx", aids)
	writeInts(chkdirprfx+"structure.cids.indx", cids)
	writeInts(chkdirprfx+"structure.ds.indx", ds)
	writeInts(chkdirprfx+"structure.upcnt.indx", upcnt)
	writeFloats(chkdirprfx+"structure.dwngrad.bil", dwngrad)
}

func (s *Structure) SaveGob(fp string) error {
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(" structure.Save %v", err)
	}
	if err := gob.NewEncoder(f).Encode(s); err != nil {
		return fmt.Errorf(" structure.Save %v", err)
	}
	f.Close()
	return nil
}

func loadGobStructure(fp string) (*Structure, error) {
	var strc Structure
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
