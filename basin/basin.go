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
	ep              map[int][366]float64
	contarea, fncid float64
	ncid            int
}

type sample struct {
	bsn        hru.Basin
	gw         gwru.TMQ
	p0, p1     map[int]float64
	rill, m, n float64
}

// Run a single simulation with water balance checking
func Run(ldr *Loader, rill, m, n float64) float64 {
	frc, mdl := ldr.load(1.)
	println()
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
	b.buildEp()
	smpl := b.toSample(rill, m, n)
	for _, c := range b.cids {
		if smpl.bsn[c] == nil {
			log.Fatalln(" basin.Run() error: nil hru")
		}
	}
	return b.evalCascKineWB(&smpl, true)
}
