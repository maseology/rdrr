package prep

import (
	"fmt"
	"log"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

// BuildRTR returns (and saves) the topological routing scheme amongst sub-basins
func BuildRTR(gobDir string, strc *model.STRC, gd *grid.Definition, swsFP string) *model.RTR {

	csws, dsws := collectSWS(swsFP, gd)

	swscidxr := make(map[int][]int)
	for _, c := range strc.CIDs {
		s := csws[c]
		// if _, ok := swscidxr[s]; ok {
		// 	swscidxr[s] = append(swscidxr[s], c)
		// } else {
		// 	swscidxr[s] = []int{c}
		// }
		swscidxr[s] = append(swscidxr[s], c)
	}

	rtr := model.RTR{
		SwsCidXR: swscidxr, // ordered cids, per sws
		Sws:      csws,     // [cid]sws mapping
		Dsws:     dsws,     // downslope topological watershed routing
	}

	if err := rtr.SaveGob(gobDir + "domain.RTR.gob"); err != nil {
		log.Fatalf(" BuildRTR error: %v", err)
	}

	return &rtr
}

// collectSWS collects sws data when provided
func collectSWS(swsFP string, gd *grid.Definition) (map[int]int, map[int]int) {

	if _, ok := mmio.FileExists(swsFP); !ok {
		fmt.Println(" *** warning: no subwatershed data provided, entire model will consist of 1 sws. ***")
		return func() (map[int]int, map[int]int) {
			cs := make(map[int]int, gd.Nact)
			for _, c := range gd.Sactives {
				cs[c] = 0
			}
			return cs, map[int]int{0: -1} //, map[int][]int{0: gd.Sactives}
		}()
	}

	var gsws grid.Indx
	gsws.LoadGDef(gd)
	gsws.New(swsFP, false)
	cs := gsws.Values()
	sc := make(map[int][]int, len(gsws.UniqueValues()))
	for c, s := range cs {
		// if _, ok := sc[s]; ok {
		// 	sc[s] = append(sc[s], c)
		// } else {
		// 	sc[s] = []int{c}
		// }
		sc[s] = append(sc[s], c)
	}

	// collect topology
	var dsws map[int]int
	topoFP := mmio.RemoveExtension(swsFP) + ".topo"
	nsws := len(sc)
	if _, ok := mmio.FileExists(topoFP); ok {
		d, err := mmio.ReadCSV(topoFP)
		if err != nil {
			log.Fatalf(" Loader.readSWS: error reading %s: %v\n", topoFP, err)
		}
		// dsws = make(map[int]int, len(d)) // note: swsids not contained within dsws drain to farfield
		// for _, ln := range d {
		// 	dsws[int(ln[1])] = int(ln[2]) // linkID,upstream_swsID,downstream_swsID
		// }
		dsws = make(map[int]int, nsws) // note: swsids not contained within dsws drain to farfield
		for _, ln := range d {
			if _, ok := sc[int(ln[1])]; ok {
				if _, ok := sc[int(ln[2])]; ok {
					dsws[int(ln[1])] = int(ln[2]) // linkID,upstream_swsID,downstream_swsID
				}
			}
		}
	} else {
		// fmt.Printf(" warning: sws topology (*.topo) not found\n")
		log.Fatalf(" BuildRTR error: sws topology (*.topo) not found: %s", topoFP)
	}

	return cs, dsws
}
