package basin

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"
)

// FORC holds forcing data
type FORC struct {
	T           []time.Time   // [date ID]
	D           [][][]float64 // D [date ID][location ID][type ID] or [cell ID][date ID][type ID]
	XR          map[int]int   // mapping of model grid cell to met grid cell
	IntervalSec float64
	mt          []int
	q0, qs      float64
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

func gwsink(sta string) float64 {
	d := map[string]float64{
		"02EC021": .0005,
		"02ED030": .00025,
		"02HB020": .0005,
		"02HC056": .0005,
		"02HC005": .00025,
	}
	if v, ok := d[sta]; ok {
		return v
	}
	return 0.
}
