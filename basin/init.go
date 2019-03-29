package basin

import (
	"math"

	"github.com/maseology/goHydro/hru"
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

func (b *Basin) toSample(rill, m, f1 float64) sample {
	h := make(map[int]*hru.HRU, b.ncid)
	for i, v := range b.mdl.b {
		hnew := *v
		hnew.Reset()
		h[i] = &hnew
	}
	return sample{
		bsn:  h,
		gw:   b.mdl.g.Clone(m),
		fc:   b.buildFc(f1),
		rill: rill,
		m:    m,
	}
}
