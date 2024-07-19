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
	Dwnslope        []float64 // downslope angle (beta, radians)
	Nc              int       // number of cells, groundwater zones
}

func (s *Structure) Checkandprint(chkdirprfx string) {

	// output
	mx := make(map[int]int, s.Nc)
	for i, c := range s.Cids {
		mx[c] = i
	}

	aid, cid, ads, nus, upcnt := s.GD.NullInt32(-9999), s.GD.NullInt32(-9999), s.GD.NullInt32(-9999), s.GD.NullInt32(-9999), s.GD.NullInt32(-9999)
	dwnslp := s.GD.NullArray(-9999.)
	for _, c := range s.GD.Sactives {
		nus[c] = 0
	}
	for _, c := range s.GD.Sactives {
		if i, ok := mx[c]; ok {
			if s.Cids[i] != c {
				panic("structure.Checkandprint cell ID error")
			}
			cid[c] = int32(s.Cids[i])
			aid[c] = int32(i)
			ads[c] = int32(s.Ds[i])
			upcnt[c] = int32(s.Upcnt[i])
			dwnslp[c] = s.Dwnslope[i]
			if s.Ds[i] >= 0 {
				nus[s.Cids[s.Ds[i]]]++
			}
		}
	}

	writeInts(s.GD, chkdirprfx+"structure.aid.bil", aid)           // ordered/topologically-sorted cell ID
	writeInts(s.GD, chkdirprfx+"structure.ads.bil", ads)           // down-slope cell array index
	writeInts(s.GD, chkdirprfx+"structure.nus.bil", nus)           // number of (upslope) cells contributing runoff to current cell
	writeInts(s.GD, chkdirprfx+"structure.cid.bil", cid)           // grid cell ID
	writeInts(s.GD, chkdirprfx+"structure.upcnt.bil", upcnt)       // count of upslope/contributing area cells
	writeFloats32(s.GD, chkdirprfx+"structure.dwnslp.bil", dwnslp) // cell slope angle (radians), aka beta
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

func LoadGobStructure(fp string) (*Structure, error) {
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
