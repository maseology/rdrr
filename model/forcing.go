package model

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"
)

// FORC holds forcing data
type FORC struct {
	T           []time.Time   // [date ID]
	D           [][][]float64 // [ 0:yield; 1:Ep ][staID][DateID]
	O           [][]float64   // observed discharge (use Oxr for cross-reference)
	XR          map[int]int   // mapping of model grid cell to met grid cell
	Oxr         []int         // mapping of outlet cell ID to O[][]
	IntervalSec float64
	// mt          []int
	// q0, qs      float64
	// Name   string
}

// SaveGob FORC to gob
func (frc *FORC) SaveGob(fp string) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return fmt.Errorf(" FORC.SaveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(frc); err != nil {
		return fmt.Errorf(" FORC.SaveGob %v", err)
	}
	return nil
}

// LoadGobFORC loads
func LoadGobFORC(fp string) (*FORC, error) {
	var frc FORC
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&frc)
	if err != nil {
		return nil, err
	}
	return &frc, nil
}
