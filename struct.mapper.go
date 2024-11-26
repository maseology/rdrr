package rdrr

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/goHydro/grid"
)

type Mapper struct {
	Mx                      map[int]int
	Ilu, Isg, Igw, Icov     []int
	Ksat, Fimp, Fint, Fngwc []float64
}

func (mp *Mapper) Checkandprint(gd *grid.Definition, fnc float64, chkdirprfx string, crop bool) {

	var gd2 *grid.Definition
	xr := make(map[int]int)
	if crop {
		gd2, xr = gd.CropToActives()
	} else {
		gd2 = gd
		for _, c := range gd.Sactives {
			xr[c] = c
		}
	}

	// summarize
	fmt.Printf("   %d groundwater zones, number of cells:\n", len(mp.Fngwc))
	for ig, n := range mp.Fngwc {
		fmt.Printf("%10d%15d  (%.1f %%)\n", ig, int(n), 100*n/fnc)
	}

	// output
	ilu, isg, igw, icov := gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999)
	ksat, fimp, fint := gd2.NullArray(-9999.), gd2.NullArray(-9999.), gd2.NullArray(-9999.)
	for _, c := range gd.Sactives {
		if i, ok := mp.Mx[c]; ok {
			c2 := xr[c]
			ilu[c2] = int32(mp.Ilu[i])
			isg[c2] = int32(mp.Isg[i])
			igw[c2] = int32(mp.Igw[i])
			icov[c2] = int32(mp.Icov[i])
			ksat[c2] = mp.Ksat[i]
			fimp[c2] = mp.Fimp[i]
			fint[c2] = mp.Fint[i]
		}
	}

	writeInts(gd2, chkdirprfx+"mapper.ilu.bil", ilu)       // land use type index
	writeInts(gd2, chkdirprfx+"mapper.isg.bil", isg)       // surficial geology type index
	writeInts(gd2, chkdirprfx+"mapper.igw.bil", igw)       // groundwater reservoir index
	writeInts(gd2, chkdirprfx+"mapper.icov.bil", icov)     // canopy cover type index
	writeFloats32(gd2, chkdirprfx+"mapper.ksat.bil", ksat) // vertical percolation rates
	writeFloats32(gd2, chkdirprfx+"mapper.fimp.bil", fimp) // fraction of impervious cover
	writeFloats32(gd2, chkdirprfx+"mapper.fint.bil", fint) // interception cover factor
}

func (mp *Mapper) SaveGob(fp string) error {
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(" mapper.SaveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(mp); err != nil {
		return fmt.Errorf(" mapper.SaveGob %v", err)
	}
	f.Close()
	return nil
}

func LoadGobMapper(fp string) (*Mapper, error) {
	var mpr Mapper
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&mpr)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &mpr, nil
}
