package basin

import (
	"fmt"
	"math"
	"sync"

	"github.com/maseology/goHydro/hru"
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
	defer gwg.Done()
	mmio.WriteFloats(fmt.Sprintf("%s%d.mon", mondir, m.c), m.v)
}

type gmonitor struct{ gy, ga, gr, gg, gb []float64 }

func (g *gmonitor) print(ws []hru.HRU, pin map[int][]float64, cxr map[int]int, ds []int, fnstep float64) {
	gmu.Lock()
	defer gmu.Unlock()
	defer gwg.Done()
	my, ma, mr, mron, mrgen, mg := make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy))
	ms, msma, msrf := make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy))
	f := 365.24 * 1000. / fnstep // [mm/yr]
	for c := range cxr {
		mron[c] = 0.
	}
	for c, i := range cxr {
		my[c] = g.gy[i] * f
		ma[c] = g.ga[i] * f
		mr[c] = g.gr[i] * f
		mg[c] = (g.gg[i] - g.gb[i]) * f
		// ms[c] = ws[i].Storage() * f
		sma, srf := ws[i].Storage2()
		msma[c] = sma * f
		msrf[c] = srf * f
		ms[c] = (sma + srf) * f
		if ds[i] > -1 {
			mron[ds[i]] += g.gr[i] * f
		}
		if _, ok := pin[i]; ok {
			for _, v := range pin[i] {
				mron[c] += v * f // add external inputs
			}
		}
	}

	for c := range cxr {
		mrgen[c] = mr[c] - mron[c]
		if mg[c] < 0. {
			mrgen[c] += mg[c] // exclude runoff from groundwater discharge
		}
		if mrgen[c] < 0. {
			mrgen[c] = 0. // exclude negative runoff (caused by greater infiltrability)
		}
	}

	mw := make(map[int]float64, len(cxr))
	for c, i := range cxr {
		y, a, g, r, o, s := my[c], ma[c], mg[c], mr[c], mron[c], ms[c]
		wbal := y + o - (a + g + r + s)
		if math.Abs(wbal) > .01*y {
			fmt.Printf("cell id %d (index %d) wbal error: (wbal = %.1fmm  delSto = %.3fmm)\n", c, i, wbal, s)
		}
		mw[c] = wbal
	}

	// NOTE: wbal = yield + ron - (aet + gwe + olf + s)
	mmio.WriteRMAP(mondir+"g.yield.rmap", my, true)
	mmio.WriteRMAP(mondir+"g.aet.rmap", ma, true)
	mmio.WriteRMAP(mondir+"g.olf.rmap", mr, true)
	mmio.WriteRMAP(mondir+"g.ron.rmap", mron, true)
	mmio.WriteRMAP(mondir+"g.rgen.rmap", mrgen, true)
	mmio.WriteRMAP(mondir+"g.gwe.rmap", mg, true)
	mmio.WriteRMAP(mondir+"g.sto.rmap", ms, true)
	mmio.WriteRMAP(mondir+"g.sma.rmap", msma, true)
	mmio.WriteRMAP(mondir+"g.srf.rmap", msrf, true)
	mmio.WriteRMAP(mondir+"g.wbal.rmap", mw, true)
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
	mmio.DeleteFile(mondir + "g.sma.rmap")
	mmio.DeleteFile(mondir + "g.srf.rmap")
	mmio.DeleteFile(mondir + "g.wbal.rmap")
	// mmio.DeleteAllSubdirectories(mondir)
}

// WaitMonitors waits for all writes to complete
func WaitMonitors() {
	gwg.Wait()
}
