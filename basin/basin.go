package basin

import (
	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
)

// Basin contais multiple HRUs and forcing data to run independently
type Basin struct {
	frc             *FRC
	mdl             *MDL
	cids            []int
	contarea, fncid float64
	ncid            int
}

type sample struct {
	bsn hru.Basin
	gw  gwru.TMQ
	// tem  tem.TEM
	rill, m float64
}

func (b *Basin) toSample(rill, m float64) sample {
	h := make(map[int]*hru.HRU, b.ncid)
	for i, v := range b.mdl.b {
		hnew := *v
		hnew.Reset()
		h[i] = &hnew
	}
	return sample{
		bsn:  h,
		gw:   b.mdl.g.Clone(m),
		rill: rill,
		m:    m,
	}
}

// Run a single simulation with water balance checking
func Run(ldr *Loader, rill, m float64) float64 {
	frc, mdl := ldr.load(1.)
	cids := mdl.t.ContributingAreaIDs(ldr.outlet)
	ncid := len(cids)
	fncid := float64(ncid)
	b := Basin{
		frc:      &frc,
		mdl:      &mdl,
		cids:     cids,
		ncid:     ncid,
		fncid:    fncid,
		contarea: mdl.a * fncid, // basin contributing area [m²]
	}
	smpl := b.toSample(rill, m)
	return b.evalWB(&smpl, true)
}
