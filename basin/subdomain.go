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

	// rarely used?
	if frc == nil {
		log.Fatalf(" domain.newSubDomain error: no forcing data provided")
	}
	cids, ds := d.strc.t.DownslopeContributingAreaIDs(outlet)
	strms := buildStreams(d.strc, cids)
	newRTR, swsord, _ := d.rtr.subset(d.strc.t, cids, strms, outlet)
	frc.subset(cids)
	ncid := len(cids)
	fncid := float64(ncid)

	for _, c := range cids {
		if p, ok := d.strc.t.TEC[c]; ok {
			if p.G <= 0. {
				fmt.Printf(" domain.newSubDomain warning: slope at cell %d was found to be %v, reset to 0.0001.", c, p.G)
				t := d.strc.t.TEC[c]
				t.G = 0.0001
				t.A = 0.
				d.strc.t.TEC[c] = t
			}
		} else {
			log.Fatalf(" domain.newSubDomain error: no topographic info available for cell %d", c)
		}
	}

	// cktopo := make(map[int]bool, len(cids))
	// for _, i := range cids {
	// 	if _, ok := cktopo[i]; ok {
	// 		log.Fatalf(" domain.newSubDomain error: cell %d occured more than once, possible cycle", i)
	// 	}
	// 	if _, ok := ds[i]; !ok {
	// 		log.Fatalf(" domain.newSubDomain error: cell %d not given dowslope id", i)
	// 	}
	// 	if _, ok := cktopo[ds[i]]; ok {
	// 		log.Fatalf(" domain.newSubDomain error: cell %d out of topological order", i)
	// 	}
	// 	cktopo[i] = true
	// }

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
		contarea: d.strc.a * fncid, // basin contributing area [m²]
		cid0:     outlet,
		gwsink:   frc.Qs,
	}
	// b.buildStreams(strmkm2)
	return b
}

func (d *domain) noSubDomain(frc *FORC) subdomain {
	if frc == nil {
		log.Fatalf(" domain.newSubDomain error: no forcing data provided")
	}
	cids0, ds := d.strc.t.DownslopeContributingAreaIDs(-1)
	cids := make([]int, d.gd.Na)
	icid := 0
	for _, cid := range cids0 {
		if _, ok := d.rtr.sws[cid]; ok {
			cids[icid] = cid
			icid++
			if _, ok := d.rtr.sws[ds[cid]]; !ok {
				ds[cid] = -1
			}
		} else {
			delete(ds, cid)
		}
	}

	// cid0 := cids[len(cids)-1] // assumes only one outlet
	// ds[cid0] = -1
	cid0 := -1
	strms := buildStreams(d.strc, cids)
	newRTR, swsord, _ := d.rtr.subset(d.strc.t, cids, strms, cid0)
	frc.subset(cids)
	ncid := len(cids)
	fncid := float64(ncid)

	for _, c := range cids {
		if p, ok := d.strc.t.TEC[c]; ok {
			if p.G <= 0. {
				fmt.Printf(" domain.noSubDomain warning: slope at cell %d was found to be %v, reset to 0.0001.", c, p.G)
				t := d.strc.t.TEC[c]
				t.G = 0.0001
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
		strms:    strms,
		swsord:   swsord,
		ds:       ds,
		obs:      buildObs(d, newRTR),
		ncid:     ncid,
		fncid:    fncid,
		nstrm:    len(strms),
		fnstrm:   float64(len(strms)),
		contarea: d.strc.a * fncid, // basin contributing area [m²]
		cid0:     cid0,
	}
	// b.buildStreams(strmkm2)
	return b
}

func buildStreams(strc *STRC, cids []int) []int {
	strmcthresh := int(strmkm2 * 1000. * 1000. / strc.w / strc.w) // "stream cell" threshold
	strms := []int{}
	for _, c := range cids {
		if strc.u[c] > strmcthresh {
			strms = append(strms, c)
		}
	}
	return strms
}

func buildObs(d *domain, r *RTR) map[int][]int {
	obs := make(map[int][]int, len(d.obs))
	for _, o := range d.obs {
		if s, ok := r.sws[o]; ok {
			if _, ok := obs[s]; ok {
				obs[s] = append(obs[s], o)
			} else {
				obs[s] = []int{o}
			}
		}
	}
	return obs
}

// func (b *subdomain) buildStreams(strmkm2 float64) {
// 	strmcthresh := int(strmkm2 * 1000. * 1000. / b.strc.w / b.strc.w) // "stream cell" threshold
// 	nstrm := 0
// 	b.strms = make([]int, 0)
// 	for _, c := range b.cids {
// 		if b.strc.u[c] > strmcthresh {
// 			b.strms = append(b.strms, c)
// 			nstrm++
// 		}
// 	}
// 	b.nstrm = nstrm
// 	b.fnstrm = float64(nstrm)
// }

func (b *subdomain) getForcings() (dt []time.Time, y, ep [][]float64, obs []float64, intvl int64, nstep int) {
	ns, nloc, x := b.frc.h.Nstep(), b.frc.h.Nloc(), b.frc.h.WBDCxr()
	dt, y, ep, obs = make([]time.Time, ns), make([][]float64, nloc), make([][]float64, nloc), make([]float64, ns)
	intvl, nstep = int64(b.frc.h.IntervalSec()), ns
	if nloc == 1 {
		y[0], ep[0] = make([]float64, ns), make([]float64, ns)
		// h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor for subdomain outlet cell
		for k, dt1 := range b.frc.c.T {
			dt[k] = dt1
			v := b.frc.c.D[k][0]
			// f := b.strc.f[c][d.YearDay()-1] // adjust for slope-aspect
			y[0][k] = v[x["AtmosphericYield"]]   // precipitation/atmospheric yield (rainfall + snowmelt)
			ep[0][k] = v[x["AtmosphericDemand"]] // evaporative demand
			obs[k] = v[x["UnitDischarge"]]       //* h2cms
		}
	} else {
		if b.frc.nam == "gob" {
			if _, ok := b.frc.h.WBDCxr()["UnitDischarge"]; ok {
				if len(b.frc.c.D[2]) != 1 {
					log.Fatalf("subdomain.getForcings error: only one outlet discharge currently supported")
				}
				obs = b.frc.c.D[2][0]
			} else {
				obs = []float64{}
			}
			for k, dt1 := range b.frc.c.T {
				dt[k] = dt1
			}
			y = b.frc.c.D[0]
			ep = b.frc.c.D[1]
		} else {
			log.Fatalf("subdomain.getForcings todo: forcings with multiple locations")
		}
	}
	return
}
