package basin

import (
	"log"

	mmio "github.com/maseology/mmio"
)

// subdomain carries all non-parameter data for a particular region (eg a catchment).
// Forcing variables are collected and held to be run independently for each sample.
type subdomain struct {
	frc             *FORC       // forcing data
	strc            *STRC       // structural data
	mpr             *MAPR       // land use/surficial geology mapping
	rtr             *RTR        // subwatershed topology
	ds              map[int]int // downslope cell ID
	cids, sids      []int       // cell IDs, sws IDs (topologically ordered)
	contarea, fncid float64     // contributing area [m²], (float) number of cells
	ncid, cid0      int         // number of cells, outlet cell ID
}

func (d *domain) newSubDomain(frc *FORC, outlet int) subdomain {
	if outlet < 0 {
		return d.noSubDomain(frc)
	}
	if frc == nil {
		log.Fatalf(" domain.newSubDomain error: no forcing data provided")
	}
	cids, ds := d.strc.t.DownslopeContributingAreaIDs(outlet)
	newRTR, swsids := d.rtr.subset(cids, outlet)
	frc.subset(cids)
	ncid := len(cids)
	fncid := float64(ncid)

	b := subdomain{
		frc:      frc,
		strc:     d.strc,
		mpr:      d.mpr,
		rtr:      newRTR,
		cids:     cids,
		sids:     swsids,
		ds:       ds,
		ncid:     ncid,
		fncid:    fncid,
		contarea: d.strc.a * fncid, // basin contributing area [m²]
		cid0:     outlet,
	}
	// mmio.WriteIMAP("E:/ormgp_rdrr/"+frc.nam+"_sws.imap", newRTR.sws)
	// mmio.WriteLines("E:/ormgp_rdrr/"+frc.nam+"_sws.txt", swsids)

	return b

	// // newSTRC, cids, ds := d.strc.subset(d.gd, outlet)
	// // newMAPR := d.mpr.subset(cids, outlet)
	// cids, ds := d.strc.t.DownslopeContributingAreaIDs(outlet)
	// ncid := len(cids)
	// fncid := float64(ncid)

	// var b subdomain
	// if frc == nil {
	// 	if d.frc == nil {
	// 		log.Fatalf(" domain.newSubDomain error: no forcing data provided")
	// 	}
	// 	b = subdomain{
	// 		frc:      frc.subset(cids),
	// 		strc:     d.strc, // newSTRC,
	// 		mpr:      d.mpr,  // newMAPR,
	// 		cids:     cids,
	// 		ds:       ds,
	// 		ncid:     ncid,
	// 		fncid:    fncid,
	// 		contarea: d.strc.a * fncid, // basin contributing area [m²]
	// 		cid0:     outlet,
	// 	}
	// } else {
	// 	b = subdomain{
	// 		frc:      frc,
	// 		strc:     d.strc, // newSTRC,
	// 		mpr:      d.mpr,  // newMAPR,
	// 		cids:     cids,
	// 		ds:       ds,
	// 		ncid:     ncid,
	// 		fncid:    fncid,
	// 		contarea: d.strc.a * fncid, // basin contributing area [m²]
	// 		cid0:     outlet,
	// 	}
	// }
	// // b.buildEp()

	// return b
}

func (d *domain) noSubDomain(frc *FORC) subdomain {
	// to complete *************
	cids, _ := d.strc.t.DownslopeContributingAreaIDs(-1)
	_, swsids := d.rtr.subset(cids, -1)
	mmio.WriteInts("E:/ormgp_rdrr/_sws.txt", swsids)
	mmio.WriteInts("E:/ormgp_rdrr/_cid.txt", cids)
	log.Fatalf(" domain.newSubDomain todo: outlet < 0\n")
	return subdomain{}
}
