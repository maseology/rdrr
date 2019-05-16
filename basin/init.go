package basin

import (
	"math"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/mmaths"
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

func (b *Basin) buildC0(n map[int]float64, ts float64) map[int]float64 {
	c0 := make(map[int]float64, len(b.cids))
	for _, cid := range b.cids {
		c := fiveThirds * math.Sqrt(b.mdl.t.TEC[cid].S) * ts / b.mdl.w / n[cid]
		c0[cid] = c / (1. + c)
	}
	return c0
}

func (b *Basin) buildC2(n map[int]float64, ts float64) map[int]float64 {
	c2 := make(map[int]float64, len(b.cids))
	for _, cid := range b.cids {
		c := fiveThirds * math.Sqrt(b.mdl.t.TEC[cid].S) * ts / b.mdl.w / n[cid]
		c2[cid] = 1. / (1. + c)
	}
	return c2
}

func (b *Basin) toSample(rill, m, n float64) sample {
	h := make(map[int]*hru.HRU, b.ncid)
	na := make(map[int]float64, b.ncid)
	ts := b.frc.h.IntervalSec()
	for i, v := range b.mdl.b {
		hnew := *v
		hnew.Reset()
		na[i] = n
		h[i] = &hnew
	}
	return sample{
		bsn:  h,
		gw:   b.mdl.g.Clone(m),
		p0:   b.buildC0(na, ts), // b.buildFc(f1),
		p1:   b.buildC2(na, ts),
		rill: rill,
	}
}

func (b *Basin) toSampleU(u ...float64) sample {
	// transform sample space
	rill := mmaths.LogLinearTransform(0.01, 1., u[0])
	topm := mmaths.LogLinearTransform(0.001, 10., u[1])
	dsoil := mmaths.LinearTransform(0.01, 1., u[2])
	dpsto := mmaths.LogLinearTransform(0.0001, 0.001, u[3])
	mann := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.0001, 100., u)
	}

	h := make(map[int]*hru.HRU, b.ncid)
	ksat, n := make(map[int]float64), make(map[int]float64)
	ts := b.frc.h.IntervalSec()
	for i := range b.mdl.b {
		var hnew hru.HRU
		hnew.Initialize(b.mpr.lu[i].DrnSto, b.mpr.lu[i].SrfSto, b.mpr.lu[i].Fimp, b.mpr.sg[i].Ksat, ts)
		h[i] = &hnew
		ksat[i] = b.mpr.sg[i].Ksat
		n[i] = b.mpr.lu[i].M
	}
	var g gwru.TMQ
	g.New(ksat, b.mdl.t.SubSet(b.cid0), b.mdl.w, b.mdl.g.Qo, 2*b.mdl.g.Qo, topm)
	return sample{
		bsn:  h,
		gw:   g,
		p0:   b.buildC0(n, ts), // b.buildFc(f1),
		p1:   b.buildC2(n, ts),
		rill: rill,
	}
}
