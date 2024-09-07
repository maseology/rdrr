package rdrr

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/goHydro/grid"
)

type Parameter struct {
	Zeta, Uca, Tanbeta, DepSto, Drel, Gamma []float64
}

func (par *Parameter) Checkandprint(gd *grid.Definition, mx map[int]int, igw []int, chkdirprfx string) {

	zeta, uca, tanbeta, depsto, drel, gamma := gd.NullArray(-9999.), gd.NullArray(-9999.), gd.NullArray(-9999.), gd.NullArray(-9999.), gd.NullArray(-9999.), gd.NullArray(-9999.)
	for _, c := range gd.Sactives {
		if i, ok := mx[c]; ok {
			zeta[c] = par.Zeta[i]
			uca[c] = par.Uca[i]
			tanbeta[c] = par.Tanbeta[i]
			drel[c] = par.Drel[i]
			gamma[c] = par.Gamma[igw[i]]
			depsto[c] = par.DepSto[i]
		}
	}

	writeFloats32(gd, chkdirprfx+"parameter.zeta.bil", zeta)       // soil-topographic index
	writeFloats32(gd, chkdirprfx+"parameter.uca.bil", uca)         // unit contributing area
	writeFloats32(gd, chkdirprfx+"parameter.tanbeta.bil", tanbeta) // surface gradient
	writeFloats32(gd, chkdirprfx+"parameter.drel.bil", drel)       // groundwater deficit relative to the regional mean (deltaD)
	writeFloats32(gd, chkdirprfx+"parameter.gamma.bil", gamma)     // groundwater reservoir average soil-topographic index
	writeFloats32(gd, chkdirprfx+"parameter.depsto.bil", depsto)   // depression storage

}

func (par *Parameter) SaveGob(fp string) error {
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(" mapper.SaveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(par); err != nil {
		return fmt.Errorf(" mapper.SaveGob %v", err)
	}
	f.Close()
	return nil
}

func loadGobParameter(fp string) (*Parameter, error) {
	var par Parameter
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&par)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &par, nil
}
