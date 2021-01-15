package model

import (
	"fmt"
	"log"
)

func (d *Domain) newSubDomain(frc *FORC, outlet int) subdomain {
	if frc == nil {
		log.Fatalf(" domain.newSubDomain error: no forcing data provided")
	}
	if outlet >= 0 {
		fmt.Printf(" subsetting master model to cell %d\n", outlet)
	} else if len(frc.Oxr) > 0 {
		if len(frc.Oxr) > 1 {
			fmt.Printf(" multiple outlet cells currently not supported\n")
		}
		outlet = frc.Oxr[0]
	}

	cids, ds := d.Strc.TEM.DownslopeContributingAreaIDs(outlet)
	strms, _ := buildStreams(d.Strc, cids)
	newRTR, swsord, _ := d.rtr.subset(d.Strc.TEM, cids, strms, outlet)
	ncid := len(cids)
	fncid := float64(ncid)

	for _, c := range cids {
		if p, ok := d.Strc.TEM.TEC[c]; ok {
			if p.G <= 0. {
				fmt.Printf(" domain.newSubDomain warning: slope at cell %d was found to be %v, reset to 0.0001.", c, p.G)
				t := d.Strc.TEM.TEC[c]
				t.G = 0.0001
				t.A = 0.
				d.Strc.TEM.TEC[c] = t
			}
		} else {
			log.Fatalf(" domain.newSubDomain error: no topographic info available for cell %d", c)
		}
	}

	mons := sortMonitorsSWS(d, newRTR)
	if mons == nil {
		mons = map[int][]int{outlet: {outlet}}
	}

	b := subdomain{
		frc:      frc,
		strc:     d.Strc,
		mpr:      d.mpr,
		rtr:      newRTR,
		cids:     cids,
		swsord:   swsord,
		ds:       ds,
		mon:      mons,
		ncid:     ncid,
		fncid:    fncid,
		nstrm:    len(strms),
		fnstrm:   float64(len(strms)),
		contarea: d.Strc.Acell * fncid, // basin contributing area [m²]
		cid0:     outlet,
	}
	return b
}

// buildStreams determines stream cells based on const strmkm2
func buildStreams(strc *STRC, cids []int) ([]int, int) {
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
func sortMonitorsSWS(d *Domain, r *RTR) map[int][]int {
	if d.mons == nil {
		return nil
	}
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
