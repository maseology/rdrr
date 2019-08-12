package basin

import (
	"log"
	"math"

	"github.com/maseology/goHydro/met"
	"github.com/maseology/mmaths"
)

// FORC holds forcing data
type FORC struct {
	c  met.Coll
	h  met.Header
	Q0 float64
}

func (f *FORC) subset(cids []int) *FORC {
	if f.h.Nloc() == 1 {
		f.Q0 = f.medQ()
		return f
	}
	// newFORC := FORC{
	// 	Q0: f.medQ(),
	// }
	// return &newFORC
	log.Fatalf(" FORC.subset error: unsupported met format")
	return nil
}

// approximating "baseflow when basin is fully saturated" (TOPMODEL) as median discharge
func (f *FORC) medQ() float64 {
	a, i := make([]float64, len(f.c)), 0
	for _, m := range f.c {
		v, ok := m[met.UnitDischarge]
		if ok && !math.IsNaN(v) {
			a[i] = v
			i++
		}
	}
	if i == 0 {
		log.Fatalln("FORC.medQ: forcing collection does contain met.UnitDischarge")
		return 0.
	}
	return mmaths.SliceMedian(a)
}
