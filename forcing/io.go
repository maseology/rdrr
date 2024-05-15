package forcing

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
)

func (frc *Forcing) SaveGobForcing(fp string) error {
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(" forcing.saveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(frc); err != nil {
		return fmt.Errorf(" forcing.saveGob %v", err)
	}
	f.Close()
	return nil
}

func LoadGobForcing(fp string) (*Forcing, error) {
	var frc Forcing
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&frc)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &frc, nil
}

func (frc *Forcing) ToBil(gd *grid.Definition, gcids []int, scids [][]int, chkdirprfx string) {
	println(" > printing forcing rasters..")

	mya, mpe := make(map[int]float64, len(scids)), make(map[int]float64, len(scids))
	f := 365.24 * 4. / float64(len(frc.T))
	for i := range scids {
		for j := range frc.T {
			mya[i] += frc.Ya[i][j]
			mpe[i] += frc.Ea[i][j]
		}
		mya[i] *= f
		mpe[i] *= f
	}

	sya, spe := gd.NullArray(-9999.), gd.NullArray(-9999.)
	for i, cids := range scids {
		for _, a := range cids {
			c := gcids[a]
			sya[c] = mya[i] * 1000.
			spe[c] = mpe[i] * 1000.
		}
	}

	writeBil32(gd, chkdirprfx+"forcing.sya.bil", sya) // mean precipitation/atmospheric yeild (mm/yr)
	writeBil32(gd, chkdirprfx+"forcing.spe.bil", spe) // mean potential evaporation (mm/yr)
}

func writeBil32(gd *grid.Definition, fp string, f []float64) {
	f32 := func() []float32 {
		o := make([]float32, len(f))
		for i, v := range f {
			o[i] = float32(v)
		}
		return o
	}()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, f32)
	os.WriteFile(fp, buf.Bytes(), 0644)
	gd.ToHDRfloat(mmio.RemoveExtension(fp)+".hdr", 1, 32)
}
