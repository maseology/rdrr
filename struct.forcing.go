package rdrr

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"
)

const intvl = 86400 / 4

type Forcing struct {
	T           []time.Time // [date ID]
	Ya, Ea      [][]float64 // [staID][DateID] atmospheric exchange terms
	IntervalSec float64
}

func (frc *Forcing) CheckAndPrint() {
	fmt.Println("Forcing summary:")
	nt := len(frc.T)
	fmt.Printf(" %v to %v, 6-hourly (%d timesteps)\n", frc.T[0], frc.T[nt-1], nt)
	nsta := len(frc.Ya)
	fmt.Printf(" model timestep interval: %ds, %d stations\n", int64(frc.IntervalSec), nsta)

	sy, se := 0., 0.
	for i := 0; i < nsta; i++ {
		for j := range frc.T {
			sy += frc.Ya[i][j]
			se += frc.Ea[i][j]
		}
	}
	sy *= 365.24 * 4. / float64(nt) / float64(nsta)
	se *= 365.24 * 4. / float64(nt) / float64(nsta)
	fmt.Printf(" totals (/yr): Ya: %.5f   Ea: %.5f\n", sy, se)
}

func (frc *Forcing) saveGob(fp string) error {
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
