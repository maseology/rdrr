package rdrr

import "fmt"

func (w *Subwatershed) remapLakes(mp *Mapper, lakfrac float64) {
	if lakfrac <= 0. {
		return
	}
	var lsids []int
	for sid, cids := range w.Scis {
		nlak, nwl := 0, 0
		for _, c := range cids {
			if mp.Ilu[c] == Lake || mp.Ilu[c] == Waterbody {
				nlak++
			} else if mp.Ilu[c] == Swamp || mp.Ilu[c] == Marsh || mp.Ilu[c] == Wetland {
				nwl++
			}
		}
		if nlak > nwl && float64(nlak+nwl)/float64(len(cids)) > lakfrac {
			lsids = append(lsids, sid)
		}
	}
	if len(lsids) > 0 {
		for _, i := range lsids {
			w.Islake[i] = true
		}
		fmt.Printf("   %d subwatersheds mapped as lakes\n", len(lsids))
	}
}
