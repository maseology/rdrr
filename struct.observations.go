package rdrr

import (
	"time"
)

// OBS holds forcing data
type Observations struct {
	Td              []time.Time // [date ID]
	Oq              [][]float64 // observed discharge (use Oxr for cross-reference)
	Oqxr, Mons, txr []int       // mapping of outlet cell ID to Oq[][]; other cell IDs to montior; month [1,12] cross-reference
	cellarea        float64     // (uniform) cell area and timestep
}

// ToDaily imports hyd [m/timestep]
func (obs *Observations) ToDaily(dat []float64) []float64 {
	nt := len(obs.Td)
	o := make([]float64, nt)
	for i, v := range dat {
		o[obs.txr[i]] += v
	}
	for j := range o {
		o[j] *= obs.cellarea / 86400. // [m³/s]
	}
	return o
}
