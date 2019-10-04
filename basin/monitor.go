package basin

import (
	"fmt"
	"sync"

	"github.com/maseology/mmio"
)

var mu sync.Mutex
var mondir string

type monitor struct {
	v []float64
	c int
}

func (m *monitor) print() {
	mmio.WriteFloats(fmt.Sprintf("%s%d.mon", mondir, m.c), m.v)
}

type gmonitor struct{ gy, ga, gr, gg, gd, gl []float64 }

func (g *gmonitor) print(xr map[int]int, fnstep float64) {
	mu.Lock()
	defer mu.Unlock()
	my, ma, mr, mg, md, ml := make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy)), make(map[int]float64, len(g.gy))
	f := 365.24 * 1000. / fnstep
	for c, i := range xr {
		my[c] = g.gy[i] * f
		ma[c] = g.ga[i] * f
		mr[c] = g.gr[i] * f
		mg[c] = g.gg[i] * f
		md[c] = g.gd[i] * f
		ml[c] = g.gl[i] * f
	}
	mmio.WriteRMAP(mondir+"g.yield.rmap", my, true)
	mmio.WriteRMAP(mondir+"g.aet.rmap", ma, true)
	mmio.WriteRMAP(mondir+"g.olf.rmap", mr, true)
	mmio.WriteRMAP(mondir+"g.ngwe.rmap", mg, true)
	mmio.WriteRMAP(mondir+"g.gwd.rmap", md, true)
	mmio.WriteRMAP(mondir+"g.sto.rmap", ml, true)
}

// DeleteMonitors deletes monitor output from previous model run
func DeleteMonitors(mdldir string) {
	mondir = mdldir
	mmio.MakeDir(mdldir)
	mmio.DeleteFile(mondir + "g.yield.rmap")
	mmio.DeleteFile(mondir + "g.aet.rmap")
	mmio.DeleteFile(mondir + "g.olf.rmap")
	mmio.DeleteFile(mondir + "g.ngwe.rmap")
	mmio.DeleteFile(mondir + "g.gwd.rmap")
	mmio.DeleteFile(mondir + "g.sto.rmap")
}
