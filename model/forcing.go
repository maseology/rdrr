package model

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"github.com/maseology/mmio"
)

// FORC holds forcing data
type FORC struct {
	T           []time.Time   // [date ID]
	D           [][][]float64 // [ 0:yield; 1:Ep ][staID][DateID]
	O           [][]float64   // observed discharge (use Oxr for cross-reference)
	XR          map[int]int   // mapping of model grid cell to met grid cell
	Oxr, mt     []int         // mapping of outlet cell ID to O[][]
	IntervalSec float64
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

// AddObservation reads csv file of "Date","Flow","Flag"
func (frc *FORC) AddObservation(csvfp string, cid int) error {
	c, err := mmio.ReadCsvDateValueFlag(csvfp)
	dd := mmio.DayDate
	if err != nil {
		return err
	}
	frc.O, frc.Oxr = make([][]float64, 1), []int{cid}
	frc.O[0] = make([]float64, len(frc.T))
	for i, t := range frc.T {
		if v, ok := c[dd(t)]; ok {
			frc.O[0][i] = v
		} else {
			frc.O[0][i] = 0.
		}
	}
	return nil
}

// // AddObservation reads csv file of "name,cid" and assumes "name.csv" is where the monitoring data are located in outDir
// func (frc *FORC) AddObservation(outDir, csvfp string, cid int) {
// 	var c map[int]postpro.ObsColl
// 	var err error
// 	c, err = postpro.GetObservations(outDir, csvfp)
// 	if err != nil {
// 		log.Fatalf(" postpro.GetObservations failed: %v", err)
// 	}
// 	for i,t := range frc.T {

// 	}
// }
