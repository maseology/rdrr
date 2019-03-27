package basin

import (
	"log"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
)

// Basin contais multiple HRUs and forcing data to run independently
type Basin struct {
	frc             *FRC
	mdl             *MDL
	cids            []int
	ds              map[int]int
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
	mdl.t = mdl.t.SubSet(ldr.outlet)
	cids, ds := mdl.t.DownslopeContributingAreaIDs(ldr.outlet) // mdl.t.ContributingAreaIDs(ldr.outlet)
	ncid := len(cids)
	fncid := float64(ncid)
	b := Basin{
		frc:      &frc,
		mdl:      &mdl,
		cids:     cids,
		ds:       ds,
		ncid:     ncid,
		fncid:    fncid,
		contarea: mdl.a * fncid, // basin contributing area [m²]
	}
	smpl := b.toSample(rill, m)
	for _, c := range b.cids {
		if smpl.bsn[c] == nil {
			log.Fatalln(" basin.Run() error: nil hru")
		}
	}
	return b.evalCascWB(&smpl, true)
}
