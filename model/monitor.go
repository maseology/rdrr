package model

import (
	"fmt"
	"math"
	"sync"

	"github.com/maseology/goHydro/hru"
	"github.com/maseology/mmio"
)

var gwg sync.WaitGroup
var gmu sync.Mutex

// var mondir string

type monitor struct {
	v []float64
	c int
}

func (m *monitor) print(mdir string) {
	defer gwg.Done()
	mmio.WriteFloats(fmt.Sprintf("%s%d.mon", mdir, m.c), m.v)
	// vv := make([]float64, len(m.v))
	// for k, v := range m.v {
	// 	vv[k] = v * h2cms
	// }
	// mmio.WriteFloats(fmt.Sprintf("%s%d.mon", mondir, m.c), vv)
}

type gmonitor struct {
	gy, ge, ga, gr, gg, gb []float64
	dir                    string
}

func (g *gmonitor) print(ws []hru.HRU, pin map[int][]float64, cxr map[int]int, ds []int, intvlSec, fnstep float64) {
	gmu.Lock()
	defer gmu.Unlock()
	defer gwg.Done()
	n := len(g.gy)
	my, me, ma, mr, mron, mrgen, mg := make(map[int]float64, n), make(map[int]float64, n), make(map[int]float64, n), make(map[int]float64, n), make(map[int]float64, n), make(map[int]float64, n), make(map[int]float64, n)
	ms, msma, msrf := make(map[int]float64, n), make(map[int]float64, n), make(map[int]float64, n)
	f := 86400. / intvlSec * 365.24 * 1000. / fnstep // [mm/yr]
	for c := range cxr {
		mron[c] = 0.
	}
	for c, i := range cxr {
		my[c] = g.gy[i] * f
		me[c] = g.ge[i] * f
		ma[c] = g.ga[i] * f
		mr[c] = g.gr[i] * f
		mg[c] = (g.gg[i] - g.gb[i]) * f
		// ms[c] = ws[i].Storage() * f
		sma, srf := ws[i].Sma.Sto, ws[i].Srf.Sto
		msma[c] = sma       //* f
		msrf[c] = srf       //* f
		ms[c] = (sma + srf) //* f
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
		y, a, g, r, o, s := my[c], ma[c], mg[c], mr[c], mron[c], ms[c]*f
		wbal := y + o - (a + g + r + s)
		if math.Abs(wbal) > .01*y {
			fmt.Printf("cell id %d (index %d) wbal error: (wbal = %.1fmm  delSto = %.3fmm)\n", c, i, wbal, s)
		}
		mw[c] = wbal
	}

	// NOTE: wbal = yield + ron - (aet + gwe + olf + s)
	mmio.WriteRMAP(g.dir+"g.yield.rmap", my, true)
	mmio.WriteRMAP(g.dir+"g.ep.rmap", me, true)
	mmio.WriteRMAP(g.dir+"g.aet.rmap", ma, true)
	mmio.WriteRMAP(g.dir+"g.olf.rmap", mr, true)
	mmio.WriteRMAP(g.dir+"g.ron.rmap", mron, true)
	mmio.WriteRMAP(g.dir+"g.rgen.rmap", mrgen, true)
	mmio.WriteRMAP(g.dir+"g.gwe.rmap", mg, true)
	mmio.WriteRMAP(g.dir+"g.sto.rmap", ms, true)
	mmio.WriteRMAP(g.dir+"g.sma.rmap", msma, true)
	mmio.WriteRMAP(g.dir+"g.srf.rmap", msrf, true)
	mmio.WriteRMAP(g.dir+"g.wbal.rmap", mw, true)
}

// DeleteMonitors deletes monitor output from previous model run
func DeleteMonitors(mdldir string) {
	// mondir = mdldir
	mmio.MakeDir(mdldir)
	mmio.DeleteFile(mdldir + "g.yield.rmap")
	mmio.DeleteFile(mdldir + "g.ep.rmap")
	mmio.DeleteFile(mdldir + "g.aet.rmap")
	mmio.DeleteFile(mdldir + "g.olf.rmap")
	mmio.DeleteFile(mdldir + "g.ron.rmap")
	mmio.DeleteFile(mdldir + "g.rgen.rmap")
	mmio.DeleteFile(mdldir + "g.gwe.rmap")
	mmio.DeleteFile(mdldir + "g.sto.rmap")
	mmio.DeleteFile(mdldir + "g.sma.rmap")
	mmio.DeleteFile(mdldir + "g.srf.rmap")
	mmio.DeleteFile(mdldir + "g.wbal.rmap")
	// mmio.DeleteAllSubdirectories(mdldir)
}

// WaitMonitors waits for all writes to complete
func WaitMonitors() {
	gwg.Wait()
}
