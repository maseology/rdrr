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
	t   []temporal
	x   map[int]int // mapping of model grid cell to met grid cell
	Q0  float64
	nam string
}

type temporal struct{ doy, mt int }

func (f *FORC) subset(cids []int) {
	if f.h.Nloc() == 1 {
		f.Q0 = f.medQ()
	} else {
		f.Q0 = avgRch
	}
	return
}

// approximating "baseflow when basin is fully saturated" (TOPMODEL) as median discharge
func (f *FORC) medQ() float64 {
	x := f.h.WBDCxr()
	if _, ok := x["UnitDischarge"]; ok {
		a, i := make([]float64, len(f.c.T)), 0
		for _, m := range f.c.D {
			v := m[0][x["UnitDischarge"]]
			if !math.IsNaN(v) {
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
	return math.NaN()
}

func (f *FORC) get(dtb, dte time.Time, col int) []float64 {
	_, fdte, intvl := f.h.BeginEndInterval() // time step interval [s]
	n := int(dte.Add(time.Second*time.Duration(intvl)).Sub(dtb).Seconds() / float64(intvl))
	fout, ii := make([]float64, n), 0
	for i, dt := range f.c.T {
		if dt.Before(dtb) {
			continue
		}
		if dt.After(dte) {
			fout[ii] = math.NaN()
		} else {
			fout[ii] = f.c.D[i][0][col]
		}
		ii++
	}
	if fdte.Before(dte) {
		for dt := fdte.Add(time.Second * time.Duration(intvl)); !dt.After(dte); dt = dt.Add(time.Second * time.Duration(intvl)) {
			fout[ii] = math.NaN()
			ii++
		}
	}
	return fout
}

// func (f *FORC) trimFrc(nYrs int) (nstep int, dtb, dte time.Time, intvl int64) {
// 	nstep = f.h.Nstep()                      // number of time steps
// 	dtb, dte, intvl = f.h.BeginEndInterval() // start date, end date, time step interval [s]
// 	if nYrs > 0 {
// 		dur, durx := dte.Sub(dtb), time.Duration(nYrs*365*86400)*time.Second
// 		if dur > durx {
// 			dtb = dte.Add(-durx)
// 			nstep = int(dte.Add(time.Second*time.Duration(intvl)).Sub(dtb).Seconds() / float64(intvl))
// 		}
// 	}
// 	return
// }
