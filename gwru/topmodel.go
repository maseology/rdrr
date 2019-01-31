package gwru

import (
	"math"
)

// TOPMODEL struct
type TOPMODEL struct {
	ti, Di           map[int]float64
	g, dm, qo, m, ca float64
}

// Update state. input g: total basin average recharge per time step [m]
// returns baseflow
func (t *TOPMODEL) Update(g float64) float64 {
	// returns baseflow
	t.dm = 0.
	for _, v := range t.Di {
		t.dm += v
	}
	t.dm /= float64(len(t.Di))
	t.dm -= g

	qb := t.qo * math.Exp(-t.dm/t.m)
	t.dm += qb / t.ca

	t.updateDeficits()
	return qb
}

func (t *TOPMODEL) updateDeficits() {
	for i, v := range t.ti {
		t.Di[i] = t.dm + t.m*(t.g-v)
	}
}
