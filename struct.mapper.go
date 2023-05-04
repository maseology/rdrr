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
	Ksat, Fimp, Ifct, Fngwc []float64
}

func (mp *Mapper) checkandprint(gd *grid.Definition, fnc float64, chkdirprfx string) {

	// summarize
	fmt.Printf("%d groundwater zones, number of cells:\n", len(mp.Fngwc))
	for ig, n := range mp.Fngwc {
		fmt.Printf("%10d%15d  (%d %%)\n", ig, int(n), int(100*n/fnc))
	}

	// output
	aid, ilu, isg, igw, icov := gd.NullInt32(-9999), gd.NullInt32(-9999), gd.NullInt32(-9999), gd.NullInt32(-9999), gd.NullInt32(-9999)
	ksat, fimp, ifct := gd.NullArray(-9999.), gd.NullArray(-9999.), gd.NullArray(-9999.)
	for _, c := range gd.Sactives {
		if i, ok := mp.Mx[c]; ok {
			aid[c] = int32(i)
			ilu[c] = int32(mp.Ilu[i])
			isg[c] = int32(mp.Isg[i])
			igw[c] = int32(mp.Igw[i])
			icov[c] = int32(mp.Icov[i])
			ksat[c] = mp.Ksat[i]
			fimp[c] = mp.Fimp[i]
			ifct[c] = mp.Ifct[i]
		}
	}

	writeInts(chkdirprfx+"mapper.aid.indx", aid)
	writeInts(chkdirprfx+"mapper.ilu.indx", ilu)
	writeInts(chkdirprfx+"mapper.isg.indx", isg)
	writeInts(chkdirprfx+"mapper.igw.indx", igw)
	writeInts(chkdirprfx+"mapper.icov.indx", icov)
	writeFloats(chkdirprfx+"mapper.ksat.bil", ksat)
	writeFloats(chkdirprfx+"mapper.fimp.bil", fimp)
	writeFloats(chkdirprfx+"mapper.ifct.bil", ifct)
}

func (mp *Mapper) saveGob(fp string) error {
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
