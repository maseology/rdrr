package basin

import (
	"fmt"
	"log"
	"time"

	"github.com/maseology/mmaths"
	mmio "github.com/maseology/mmio"
)

// subdomain carries all non-parameter data for a particular region (eg a catchment).
// Forcing variables are collected and held to be run independently for each sample.
type subdomain struct {
	frc                             *FORC         // forcing data
	strc                            *STRC         // structural data
	mpr                             *MAPR         // land use/surficial geology mapping
	rtr                             *RTR          // subwatershed topology and mapping
	obs                             map[int][]int // sws{[]obs-cid}
	ds                              map[int]int   // downslope cell ID
	swsord                          [][]int       // sws IDs (topologically ordered, concurrent safe)
	cids, strms                     []int         // cell IDs (topologically ordered); stream cell IDs
	contarea, fncid, fnstrm, gwsink float64       // contributing area [m²], (float) number of cells
	ncid, nstrm, cid0               int           // number of cells, number of stream cells, outlet cell ID
	mdldir                          string        // model directory
}

func (b *subdomain) print() {
	fmt.Println("Land Use proportions")
	mLU := make(map[int]int, 10)
	for _, i := range b.cids {
		v := b.mpr.ilu[i]
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

	fmt.Println("\nSurficial Geology proportions")
	mSG := make(map[int]int, 10)
	for _, i := range b.cids {
		v := b.mpr.isg[i]
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
}

func (b *subdomain) write(dir string) error {
	b.rtr.write(dir + "b.rtr.")
	b.mpr.writeSubset(dir+"b.mpr.", b.cids)
	ucnt, strm := make(map[int]float64, b.ncid), make(map[int]bool, b.nstrm)
	for _, c := range b.cids {
		ucnt[c] = float64(b.strc.UpCnt[c])
		if b.strc.UpCnt[c] > 400 {
			strm[c] = true
		}
	}
	mmio.WriteRMAP(dir+"b.strc.t.upcnt.rmap", ucnt, false)
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
	return nil
}

func (d *domain) newSubDomain(frc *FORC, outlet int) subdomain {
	if frc == nil {
		log.Fatalf(" domain.newSubDomain error: no forcing data provided")
	}
	if outlet >= 0 {
		fmt.Println("subsetting master model")
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
				fmt.Printf(" domain.noSubDomain warning: slope at cell %d was found to be %v, reset to 0.0001.", c, p.G)
				t := d.strc.TEM.TEC[c]
				t.G = 0.0001
				t.A = 0.
				d.strc.TEM.TEC[c] = t
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
		strms:    strms,
		swsord:   swsord,
		ds:       ds,
		obs:      buildObs(d, newRTR),
		ncid:     ncid,
		fncid:    fncid,
		nstrm:    len(strms),
		fnstrm:   float64(len(strms)),
		contarea: d.strc.Acell * fncid, // basin contributing area [m²]
		cid0:     outlet,
	}
	return b
}

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

func buildObs(d *domain, r *RTR) map[int][]int {
	obs := make(map[int][]int, len(d.obs))
	for _, o := range d.obs {
		if s, ok := r.Sws[o]; ok {
			if _, ok := obs[s]; ok {
				obs[s] = append(obs[s], o)
			} else {
				obs[s] = []int{o}
			}
		}
	}
	return obs
}

func (b *subdomain) getForcings() (dt []time.Time, y, ep [][]float64, obs []float64, intvl int64, nstep int) {
	dt = b.frc.T
	y = b.frc.D[0]
	ep = b.frc.D[1]
	obs = []float64{}
	intvl = int64(b.frc.IntervalSec)
	nstep = len(b.frc.T)
	return
}
