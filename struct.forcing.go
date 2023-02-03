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

func loadGobForcing(fp string) (*Forcing, error) {
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
