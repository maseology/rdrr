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

func (s *Structure) Checkandprint(chkdirprfx string, crop bool) {

	var gd2 *grid.Definition
	xr := make(map[int]int)
	if crop {
		gd2, xr = s.GD.CropToActives()
	} else {
		gd2 = s.GD
		for _, c := range s.GD.Sactives {
			xr[c] = c
		}
	}

	// output
	mx := make(map[int]int, s.Nc)
	for i, c := range s.Cids {
		mx[c] = i
	}

	aid, cid, ads, nus, upcnt := gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999)
	dwnslp := gd2.NullArray(-9999.)
	for _, c := range gd2.Sactives {
		nus[c] = 0
	}
	for _, c := range s.GD.Sactives {
		if i, ok := mx[c]; ok {
			if s.Cids[i] != c {
				panic("structure.Checkandprint cell ID error")
			}
			c2 := xr[c]
			cid[c2] = int32(s.Cids[i])
			aid[c2] = int32(i)
			ads[c2] = int32(s.Ds[i])
			upcnt[c2] = int32(s.Upcnt[i])
			dwnslp[c2] = s.Dwnslope[i]
			if s.Ds[i] >= 0 {
				nus[xr[s.Cids[s.Ds[i]]]]++
			}
		}
	}

	writeInts(gd2, chkdirprfx+"structure.aid.bil", aid)           // ordered/topologically-sorted cell ID
	writeInts(gd2, chkdirprfx+"structure.ads.bil", ads)           // down-slope cell array index
	writeInts(gd2, chkdirprfx+"structure.nus.bil", nus)           // number of (upslope) cells contributing runoff to current cell
	writeInts(gd2, chkdirprfx+"structure.cid.bil", cid)           // grid cell ID
	writeInts(gd2, chkdirprfx+"structure.upcnt.bil", upcnt)       // count of upslope/contributing area cells
	writeFloats32(gd2, chkdirprfx+"structure.dwnslp.bil", dwnslp) // cell slope (rise/run)
	writeFloatsTxt(chkdirprfx+"structure.dwnslp.csv", s.Dwnslope) // cell slope (rise/run)
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
