package model

// subdomain carries all structural (non-parameter) data for a particular region (e.g. a catchment).
type subdomain struct {
	frc                             *FORC         // forcing data
	strc                            *STRC         // structural data
	mpr                             *MAPR         // land use/surficial geology mapping
	rtr                             *RTR          // subwatershed topology and mapping
	mon                             map[int][]int // monitor locations: sws{[]obs-cid}
	ds                              map[int]int   // downslope cell ID
	swsord                          [][]int       // sws IDs (topologically ordered, concurrent safe)
	cids                            []int         // cell IDs (topologically ordered)
	contarea, fncid, fnstrm, gwsink float64       // contributing area [m²], (float) number of cells
	ncid, nstrm, cid0               int           // number of cells, number of stream cells, outlet cell ID
}

// func (b *subdomain) getForcings() (dt []time.Time, y, ep [][]float64, obs []float64, intvl int64, nstep int) {
// 	dt = b.frc.T
// 	y = b.frc.D[0]
// 	ep = b.frc.D[1]
// 	obs = []float64{}
// 	intvl = int64(b.frc.IntervalSec)
// 	nstep = len(b.frc.T)
// 	return
// }

// func gwsink(sta string) float64 {
// 	d := map[string]float64{
// 		"02EC021": .0005,
// 		"02ED030": .00025,
// 		"02HB020": .0005,
// 		"02HC056": .0005,
// 		"02HC005": .00025, // m/ts
// 		// "02HJ005": .08,    // m³/s
// 	}
// 	if v, ok := d[sta]; ok {
// 		return v
// 	}
// 	return 0.
// }
