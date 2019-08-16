package basin

import (
	"log"
	"math"
	"time"

	"github.com/maseology/goHydro/met"
	"github.com/maseology/mmaths"
)

// FORC holds forcing data
type FORC struct {
	c   met.Coll
	h   met.Header
	Q0  float64
	nam string
}

func (f *FORC) subset(cids []int) {
	if f.h.Nloc() == 1 {
		f.Q0 = f.medQ()
	} else {
		// newFORC := FORC{
		// 	Q0: f.medQ(),
		// }
		// return &newFORC
		log.Fatalf(" FORC.subset error: unsupported met format")
	}
	return
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

func (f *FORC) trimFrc(nYrs int) (nstep int, dtb, dte time.Time, intvl int64) {
	nstep = f.h.Nstep()                      // number of time steps
	dtb, dte, intvl = f.h.BeginEndInterval() // start date, end date, time step interval [s]
	if nYrs > 0 {
		dur, durx := dte.Sub(dtb), time.Duration(nYrs*365*86400)*time.Second
		if dur > durx {
			dtb = dte.Add(-durx)
			nstep = int(dte.Add(time.Second*time.Duration(intvl)).Sub(dtb).Seconds() / float64(intvl))
		}
	}
	return
}
