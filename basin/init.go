package basin

import (
	"math"

	"github.com/maseology/goHydro/hru"
)

const (
	twoThirds  = 2. / 3.
	fiveThirds = 5. / 3.
)

func (b *Basin) buildEp() {
	b.ep = make(map[int][366]float64, len(b.cids))
	for _, c := range b.cids {
		var epc [366]float64
		for j := 0; j < 366; j++ {
			epc[j] = sinEp(j) * b.mdl.f[c][j]
		}
		b.ep[c] = epc
	}
}

func (b *Basin) buildFc(f1 float64) map[int]float64 {
	fc := make(map[int]float64, len(b.cids))
	for _, c := range b.cids {
		fc[c] = math.Min(f1*b.mdl.t.TEC[c].S, 1.)
	}
	return fc
}

func (b *Basin) buildC0(n, ts float64) map[int]float64 {
	c0 := make(map[int]float64, len(b.cids))
	for _, cid := range b.cids {
		c := fiveThirds * math.Sqrt(b.mdl.t.TEC[cid].S) * ts / b.mdl.w / n
		c0[cid] = c / (1. + c)
	}
	return c0
}

func (b *Basin) buildC2(n, ts float64) map[int]float64 {
	c2 := make(map[int]float64, len(b.cids))
	for _, cid := range b.cids {
		c := fiveThirds * math.Sqrt(b.mdl.t.TEC[cid].S) * ts / b.mdl.w / n
		c2[cid] = 1. / (1. + c)
	}
	return c2
}

func (b *Basin) toSample(rill, m, n float64) sample {
	h := make(map[int]*hru.HRU, b.ncid)
	ts := b.frc.h.IntervalSec()
	for i, v := range b.mdl.b {
		hnew := *v
		hnew.Reset()
		h[i] = &hnew
	}
	return sample{
		bsn:  h,
		gw:   b.mdl.g.Clone(m),
		p0:   b.buildC0(n, ts), // b.buildFc(f1),
		p1:   b.buildC2(n, ts),
		rill: rill,
		m:    m,
		n:    n,
	}
}
