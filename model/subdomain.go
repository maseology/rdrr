package model

import (
	"fmt"
	"log"

	"github.com/maseology/mmaths"
	mmio "github.com/maseology/mmio"
)

// subdomain carries all structural (non-parameter) data for a particular region (e.g. a catchment).
type subdomain struct {
	frc    *FORC         // forcing data
	strc   *STRC         // structural data
	mpr    *MAPR         // land use/surficial geology mapping
	rtr    *RTR          // subwatershed topology and mapping
	mon    map[int][]int // monitor locations: sws{[]obs-cid}
	ds     map[int]int   // downslope cell ID
	swsord [][]int       // sws IDs (topologically ordered, concurrent safe)
	obs    []float64     // observed data set used for optimization
	cids   []int         // cell IDs (topologically ordered)
	// cids, strms                     []int         // cell IDs (topologically ordered); stream cell IDs
	contarea, fncid, fnstrm, gwsink float64 // contributing area [m²], (float) number of cells
	ncid, nstrm, cid0               int     // number of cells, number of stream cells, outlet cell ID
	// mdldir                          string        // model directory
}

func (b *subdomain) print() {
	fmt.Println("\nLand Use proportions")
	mLU := make(map[int]int, 10)
	for _, i := range b.cids {
		v := b.mpr.LUx[i]
		if _, ok := mLU[v]; ok {
			mLU[v]++
		} else {
			mLU[v] = 1
		}
	}
	k, v := mmaths.SortMapInt(mLU)
	for i := len(k) - 1; i >= 0; i-- {
		fmt.Printf("%10d %10.1f%%\n", k[i], float64(v[i])*100./float64(len(b.cids)))
	}

	fmt.Println("Surficial Geology proportions")
	mSG := make(map[int]int, 10)
	for _, i := range b.cids {
		v := b.mpr.SGx[i]
		if _, ok := mSG[v]; ok {
			mSG[v]++
		} else {
			mSG[v] = 1
		}
	}
	k, v = mmaths.SortMapInt(mSG)
	for i := len(k) - 1; i >= 0; i-- {
		fmt.Printf("%10d %10.1f%%\n", k[i], float64(v[i])*100./float64(len(b.cids)))
	}
	println()
}

func (b *subdomain) write(dir string) error {
	b.rtr.write(dir + "b.rtr.")
	b.mpr.writeSubset(dir+"b.mpr.", b.cids)
	ucnt, strm := make(map[int]float64, b.ncid), make(map[int]bool, b.nstrm)
	slp := make(map[int]float64, b.ncid)
	mxr := make(map[int]int, b.ncid)
	for _, c := range b.cids {
		ucnt[c] = float64(b.strc.UpCnt[c])
		slp[c] = b.strc.TEM.TEC[c].G
		mxr[c] = b.frc.XR[c]
		if b.strc.UpCnt[c] > 400 {
			strm[c] = true
		}
	}
	mmio.WriteRMAP(dir+"b.strc.t.upcnt.rmap", ucnt, false)
	mmio.WriteRMAP(dir+"b.strc.t.grad.rmap", slp, false)
	mmio.WriteIMAP(dir+"b.frc.mxr.imap", mxr)
	strmca := make(map[int]int, b.ncid)
	for k := range strm {
		strmca[k] = k
		for _, c := range b.strc.TEM.UpIDs(k) {
			if _, ok := strm[c]; !ok {
				for _, c2 := range b.strc.TEM.ContributingAreaIDs(c) {
					strmca[c2] = k
				}
			}
		}
	}
	mmio.WriteIMAP(dir+"b.strc.t.strmca.imap", strmca)

	// func() { // print summary
	// 	// revxr, _ := mmio.InvertMap(b.frc.XR)
	// 	y, ep := b.frc.D[0], b.frc.D[1]
	// 	nsta := len(y)
	// 	if nsta != len(ep) {
	// 		log.Fatalln(" subdomain.write print summary error 1")
	// 	}
	// 	f := 86400. / b.frc.IntervalSec * 365.24 * 1000. / float64(len(b.frc.T))
	// 	for i := 0; i < nsta; i++ {
	// 		ss, ee := 0., 0.
	// 		for k := range b.frc.T {
	// 			ss += y[i][k]
	// 			ee += ep[i][k]
	// 		}
	// 		fmt.Printf("%d: sy: %.1f  se: %.1f\n", i, ss*f, ee*f) // mm/yr
	// 	}
	// }()

	return nil
}

func (d *domain) newSubDomain(frc *FORC, outlet int) subdomain {
	if frc == nil {
		log.Fatalf(" domain.newSubDomain error: no forcing data provided")
	}
	if outlet >= 0 {
		fmt.Println(" subsetting master model")
	}

	cids, ds := d.strc.TEM.DownslopeContributingAreaIDs(outlet)
	// cids := make([]int, d.gd.Na)
	// icid := 0
	// for _, cid := range cids0 {
	// 	if _, ok := d.rtr.sws[cid]; ok {
	// 		cids[icid] = cid
	// 		icid++
	// 		if _, ok := d.rtr.sws[ds[cid]]; !ok {
	// 			ds[cid] = -1 // farfield
	// 		}
	// 	} else {
	// 		delete(ds, cid)
	// 	}
	// }

	strms, _ := BuildStreams(d.strc, cids)
	newRTR, swsord, _ := d.rtr.subset(d.strc.TEM, cids, strms, outlet)
	ncid := len(cids)
	fncid := float64(ncid)

	for _, c := range cids {
		if p, ok := d.strc.TEM.TEC[c]; ok {
			if p.G <= 0. {
				fmt.Printf(" domain.newSubDomain warning: slope at cell %d was found to be %v, reset to 0.0001.", c, p.G)
				t := d.strc.TEM.TEC[c]
				t.G = 0.0001
				t.A = 0.
				d.strc.TEM.TEC[c] = t
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
		mon:      sortMonitorsSWS(d, newRTR),
		obs:      []float64{},
		ncid:     ncid,
		fncid:    fncid,
		nstrm:    len(strms),
		fnstrm:   float64(len(strms)),
		contarea: d.strc.Acell * fncid, // basin contributing area [m²]
		cid0:     outlet,
		// strms:    strms,
	}
	return b
}

// BuildStreams determines stream cells based on const strmkm2
func BuildStreams(strc *STRC, cids []int) ([]int, int) {
	strmcthresh := int(strmkm2 * 1000. * 1000. / strc.Acell) // "stream cell" threshold
	strms, nstrm := []int{}, 0
	for _, c := range cids {
		if strc.UpCnt[c] > strmcthresh {
			strms = append(strms, c)
			nstrm++
		}
	}
	return strms, nstrm
}

// sortMonitorsSWS sorts observation cell IDs by SWS, where d.obs ([]int{cellID}) --> b.obs (map[sid][]int{cellID})
func sortMonitorsSWS(d *domain, r *RTR) map[int][]int {
	m := make(map[int][]int, len(d.mons))
	for _, o := range d.mons {
		if s, ok := r.Sws[o]; ok {
			if _, ok := m[s]; ok {
				m[s] = append(m[s], o)
			} else {
				m[s] = []int{o}
			}
		}
	}
	return m
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