package rdrr

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/goHydro/grid"
)

type Subwatershed struct {
	Outer, Scis, Sds, Dsws [][]int
	Sid, Isws, Sgw         []int
	Fnsc                   []float64
	Ns                     int
}

func (w *Subwatershed) checkandprint(gd *grid.Definition, cids []int, fnc float64, chkdirprfx string) {

	// summarize
	fmt.Printf("%d sub-watersheds in %d rounds, ID, number of cells, computionally ordered:\n", w.Ns, len(w.Outer))
	for k, inner := range w.Outer {
		fmt.Printf("   round %d (%d)\n", k+1, len(inner))
		for _, isw := range inner {
			n := w.Fnsc[isw]
			fmt.Printf("%10d%15d%15d  (%d %%)\n", isw, w.Isws[isw], int(n), int(100*n/fnc))
		}
	}

	mx := make(map[int]int, len(cids))
	wcis := make(map[int][]int, w.Ns)
	for i, c := range cids {
		wcis[w.Sid[i]] = append(wcis[w.Sid[i]], i)
		mx[c] = i
	}

	si, sids, dsws, sgw := gd.NullInt32(-9999), gd.NullInt32(-9999), gd.NullInt32(-9999), gd.NullInt32(-9999)
	for _, c := range gd.Sactives {
		if i, ok := mx[c]; ok {
			si[c] = int32(w.Sid[i])
			sids[c] = int32(w.Isws[w.Sid[i]])
			dsws[c] = int32(w.Dsws[w.Sid[i]][0])
			sgw[c] = int32(w.Sgw[w.Sid[i]])
		}
	}

	sord := gd.NullInt32(-9999)
	for k, inner := range w.Outer {
		for _, isw := range inner {
			for _, i := range wcis[isw] {
				sord[cids[i]] = int32(k + 1)
			}
		}
	}

	writeInts(chkdirprfx+"sws.swsi.indx", si)     // zero-based index
	writeInts(chkdirprfx+"sws.swsids.indx", sids) // original index
	writeInts(chkdirprfx+"sws.sgw.indx", sgw)     // groundwater index, now projected to sws
	writeInts(chkdirprfx+"sws.dsws.indx", dsws)   // downslop sws index
	writeInts(chkdirprfx+"sws.order.indx", sord)  // computational sws ordering
}

func (ws *Subwatershed) saveGob(fp string) error {
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(" mapper.SaveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(ws); err != nil {
		return fmt.Errorf(" mapper.SaveGob %v", err)
	}
	f.Close()
	return nil
}

func loadGobSubwatershed(fp string) (*Subwatershed, error) {
	var rtr Subwatershed
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&rtr)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &rtr, nil
}
