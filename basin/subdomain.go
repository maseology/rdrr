package basin

// subdomain carries all non-parameter data for a particular region (eg a catchment).
// Forcing variables are collected and held to be run independently for each sample.
type subdomain struct {
	frc             *FORC                // forcing data
	strc            *STRC                // structural data
	mpr             *MAPR                // land use/surficial geology mapping
	ep              map[int][366]float64 // potential evaporation
	ds              map[int]int          // downslope cell ID
	cids            []int                // cell IDs (topologically ordered)
	contarea, fncid float64              // contributing area, (float) number of cells
	ncid, cid0      int                  // number of cells, outlet cell ID
}

func (d *domain) newSubDomain(outlet int) subdomain {
	newSTRC, cids, ds := d.strc.subset(outlet)
	newMAPR := d.mpr.subset(cids)
	ncid := len(cids)
	fncid := float64(ncid)

	b := subdomain{
		frc:      d.frc.subset(cids),
		strc:     newSTRC,
		mpr:      newMAPR,
		cids:     cids,
		ds:       ds,
		ncid:     ncid,
		fncid:    fncid,
		contarea: d.strc.a * fncid, // basin contributing area [m²]
		cid0:     outlet,
	}
	b.buildEp()
	return b
}

func (b *subdomain) buildEp() {
	// Sine-curve PET
	b.ep = make(map[int][366]float64, len(b.cids))
	for _, c := range b.cids {
		var epc [366]float64
		for j := 0; j < 366; j++ {
			epc[j] = sinEp(j) * b.strc.f[c][j]
		}
		b.ep[c] = epc
	}
}
