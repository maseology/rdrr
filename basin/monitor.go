package basin

import (
	"fmt"
	"sync"

	"github.com/maseology/mmio"
)

var gwg sync.WaitGroup
var gmu sync.Mutex
var mondir string

type monitor struct {
	v []float64
	c int
}

func (m *monitor) print() {
	gwg.Add(1)
	defer gwg.Done()
	mmio.WriteFloats(fmt.Sprintf("%s%d.mon", mondir, m.c), m.v)
}

type gmonitor struct{ gy, ga, gr, gg, gd, gl []float64 }

func (g *gmonitor) print(xr map[int]int, ds []int, fnstep float64) {
	gwg.Add(1)
	gmu.Lock()
	defer gmu.Unlock()
	defer gwg.Done()
	my, ma, mr, mron, mroff, mg, ml := make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy))
	f := 365.24 * 1000. / fnstep
	for c, i := range xr {
		my[c] = g.gy[i] * f
		ma[c] = g.ga[i] * f
		mr[c] = g.gr[i] * f
		mg[c] = (g.gg[i] - g.gd[i]) * f
		ml[c] = g.gl[i] * f
		if ds[i] > -1 {
			mron[ds[i]] += g.gr[i] * f
		}
	}
	for c := range xr {
		mroff[c] = mr[c] - mron[c]
		if mg[c] < 0. {
			mroff[c] += mg[c] // exclude runoff from groundwater discharge
		}
		if mroff[c] < 0. {
			mroff[c] = 0. // exclude negative runoff (caused by greater infiltrability)
		}
	}
	// NOTE: wbal = yield + ron - (aat + gwe + olf)
	mmio.WriteRMAP(mondir+"g.yield.rmap", my, true)
	mmio.WriteRMAP(mondir+"g.aet.rmap", ma, true)
	mmio.WriteRMAP(mondir+"g.olf.rmap", mr, true)
	mmio.WriteRMAP(mondir+"g.ron.rmap", mron, true)
	mmio.WriteRMAP(mondir+"g.rgen.rmap", mroff, true)
	mmio.WriteRMAP(mondir+"g.gwe.rmap", mg, true)
	mmio.WriteRMAP(mondir+"g.sto.rmap", ml, true)
}

// DeleteMonitors deletes monitor output from previous model run
func DeleteMonitors(mdldir string) {
	mondir = mdldir
	mmio.MakeDir(mdldir)
	mmio.DeleteFile(mondir + "g.yield.rmap")
	mmio.DeleteFile(mondir + "g.aet.rmap")
	mmio.DeleteFile(mondir + "g.olf.rmap")
	mmio.DeleteFile(mondir + "g.ron.rmap")
	mmio.DeleteFile(mondir + "g.rgen.rmap")
	mmio.DeleteFile(mondir + "g.gwe.rmap")
	mmio.DeleteFile(mondir + "g.sto.rmap")
}

// WaitMonitors waits for all writes to complete
func WaitMonitors() {
	gwg.Wait()
}
