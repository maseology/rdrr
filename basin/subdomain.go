package basin

import (
	"fmt"
	"log"

	mmio "github.com/maseology/mmio"
)

// subdomain carries all non-parameter data for a particular region (eg a catchment).
// Forcing variables are collected and held to be run independently for each sample.
type subdomain struct {
	frc                     *FORC       // forcing data
	strc                    *STRC       // structural data
	mpr                     *MAPR       // land use/surficial geology mapping
	rtr                     *RTR        // subwatershed topology and mapping
	ds                      map[int]int // downslope cell ID
	swsord                  [][]int     // sws IDs (topologically ordered, concurrent safe)
	cids                    []int       // cell IDs (topologically ordered)
	contarea, fncid, fnstrm float64     // contributing area [m²], (float) number of cells
	ncid, nstrm, cid0       int         // number of cells, number of stream cells, outlet cell ID
}

func (b *subdomain) print(dir string) error {
	b.rtr.print(dir + "b.rtr.")
	b.mpr.printSubset(dir+"b.mpr.", b.cids)
	ucnt, strm := make(map[int]float64, b.ncid), make(map[int]bool, b.nstrm)
	for _, c := range b.cids {
		ucnt[c] = float64(b.strc.u[c])
		if b.strc.u[c] > 400 {
			strm[c] = true
		}
	}
	mmio.WriteRMAP(dir+"b.strc.t.upcnt.rmap", ucnt, false)
	strmca := make(map[int]int, b.ncid)
	for k := range strm {
		strmca[k] = k
		for _, c := range b.strc.t.UpIDs(k) {
			if _, ok := strm[c]; !ok {
				for _, c2 := range b.strc.t.ContributingAreaIDs(c) {
					strmca[c2] = k
				}
			}
		}
	}
	mmio.WriteIMAP(dir+"b.strc.t.strmca.imap", strmca)
	return nil
}

func (d *domain) newSubDomain(frc *FORC, outlet int) subdomain {
	_, ok := d.strc.t.TEC[outlet]
	if outlet < 0 || !ok {
		return d.noSubDomain(frc)
	}
	if frc == nil {
		log.Fatalf(" domain.newSubDomain error: no forcing data provided")
	}
	cids, ds := d.strc.t.DownslopeContributingAreaIDs(outlet)
	newRTR, swsord, _ := d.rtr.subset(cids, outlet)
	frc.subset(cids)
	ncid := len(cids)
	fncid := float64(ncid)

	for _, c := range cids {
		if p, ok := d.strc.t.TEC[c]; ok {
			if p.S <= 0. {
				fmt.Printf(" domain.newSubDomain warning: slope at cell %d was found to be %v, reset to 0.0001.", c, p.S)
				t := d.strc.t.TEC[c]
				t.S = 0.0001
				t.A = 0.
				d.strc.t.TEC[c] = t
			}
		} else {
			log.Fatalf(" domain.newSubDomain error: no topographic info available for cell %d", c)
		}
	}

	b := subdomain{
		frc:      frc,
		strc:     d.strc,
		mpr:      d.mpr,
		rtr:      newRTR,
		cids:     cids,
		swsord:   swsord,
		ds:       ds,
		ncid:     ncid,
		fncid:    fncid,
		contarea: d.strc.a * fncid, // basin contributing area [m²]
		cid0:     outlet,
	}
	b.buildStreams(strmkm2)
	return b
}

func (d *domain) noSubDomain(frc *FORC) subdomain {
	if frc == nil {
		log.Fatalf(" domain.newSubDomain error: no forcing data provided")
	}
	cids, ds := d.strc.t.DownslopeContributingAreaIDs(-1)
	cid0 := cids[len(cids)-1] // assumes only one outlet
	ds[cid0] = -1
	newRTR, swsord, _ := d.rtr.subset(cids, cids[len(cids)-1]) // assumes only one outlet
	frc.subset(cids)
	ncid := len(cids)
	fncid := float64(ncid)

	for _, c := range cids {
		if p, ok := d.strc.t.TEC[c]; ok {
			if p.S <= 0. {
				fmt.Printf(" domain.noSubDomain warning: slope at cell %d was found to be %v, reset to 0.0001.", c, p.S)
				t := d.strc.t.TEC[c]
				t.S = 0.0001
				t.A = 0.
				d.strc.t.TEC[c] = t
			}
		} else {
			log.Fatalf(" domain.noSubDomain error: no topographic info available for cell %d", c)
		}
	}

	b := subdomain{
		frc:      frc,
		strc:     d.strc,
		mpr:      d.mpr,
		rtr:      newRTR,
		cids:     cids,
		swsord:   swsord,
		ds:       ds,
		ncid:     ncid,
		fncid:    fncid,
		contarea: d.strc.a * fncid, // basin contributing area [m²]
		cid0:     cid0,
	}
	b.buildStreams(strmkm2)
	return b
}

func (b *subdomain) buildStreams(strmkm2 float64) {
	strmcthresh := int(strmkm2 * 1000. * 1000. / b.strc.w / b.strc.w) // "stream cell" threshold
	nstrm := 0
	for _, c := range b.cids {
		if b.strc.u[c] > strmcthresh {
			nstrm++
		}
	}
	b.nstrm = nstrm
	b.fnstrm = float64(nstrm)
}
