package model

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"
)

// FORC holds forcing data
type FORC struct {
	T      []time.Time // [date ID]
	Ya, Ea [][]float64 // [staID][DateID] atmospheric exchange terms
	XR     map[int]int // mapping of model grid cell id to met index
	// Xt          []int       // cross reference of T-array to the input data array. (-1 means there was missing import dates.)
	IntervalSec float64
}

// SaveGob FORC to gob
func (frc *FORC) SaveGob(fp string) error {
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(" FORC.SaveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(frc); err != nil {
		return fmt.Errorf(" FORC.SaveGob %v", err)
	}
	f.Close()
	return nil
}

// LoadGobFORC loads
func LoadGobFORC(fp string) (*FORC, error) {
	var frc FORC
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
