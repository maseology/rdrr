package rdrr

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/goHydro/grid"
)

type Subwatershed struct {
	Outer, Scis, Sds [][]int
	Dsws             []SWStopo
	Sid, Isws, Sgw   []int
	Fnsc             []float64
	Islake           []bool
	Ns               int
}

type SWStopo struct{ Sid, Cid int } // receiving sws id, receiving cell id

func (w *Subwatershed) checkandprint(gd *grid.Definition, cids []int, fnc float64, chkdirprfx string) {

	// checking routing
	for _, j := range w.Dsws {
		if j.Sid > -1 {
			if j.Cid > len(w.Scis[j.Sid]) {
				panic("Subwatershed.checkandprint routing error")
			}
		}
	}

	// summarize
	fmt.Printf("   %d sub-watersheds in %d rounds, computionally ordered:\n", w.Ns, len(w.Outer))
	if len(w.Outer) > 10 {
		for k, inner := range w.Outer {
			if k < 3 || k == len(w.Outer)-1 {
				fmt.Printf("        round %d (%d)\n", k+1, len(inner))
			} else if k == 3 {
				print("         ...\n")
			}
		}
	} else {
		println("        ID          SWSID        n cells  (%%of domain)")
		for k, inner := range w.Outer {
			fmt.Printf("    round %d (%d)\n", k+1, len(inner))
			for _, isw := range inner {
				fmt.Printf("%10d%15d%15d  (%d %%)\n", isw, w.Isws[isw], int(w.Fnsc[isw]), int(100*w.Fnsc[isw]/fnc))
			}
		}
	}

	mx := make(map[int]int, len(cids))
	wcis := make(map[int][]int, w.Ns)
	for i, c := range cids {
		wcis[w.Sid[i]] = append(wcis[w.Sid[i]], i)
		mx[c] = i
	}

	si, sids, dsws, dcid, sgw := gd.NullInt32(-9999), gd.NullInt32(-9999), gd.NullInt32(-9999), gd.NullInt32(-9999), gd.NullInt32(-9999)
	hassgw := w.Sgw != nil
	for _, c := range gd.Sactives {
		if i, ok := mx[c]; ok {
			si[c] = int32(w.Sid[i])
			sids[c] = int32(w.Isws[w.Sid[i]])
			dsws[c] = int32(w.Dsws[w.Sid[i]].Sid)
			dcid[c] = int32(w.Dsws[w.Sid[i]].Cid)
			if hassgw {
				sgw[c] = int32(w.Sgw[w.Sid[i]])
			}
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

	writeInts(gd, chkdirprfx+"sws.aid.bil", si)   // zero-based index
	writeInts(gd, chkdirprfx+"sws.sid.bil", sids) // original index
	if hassgw {
		writeInts(gd, chkdirprfx+"sws.sgw.bil", sgw) // groundwater index, now projected to sws
	}
	writeInts(gd, chkdirprfx+"sws.dsws.bil", dsws)  // downslope sws index
	writeInts(gd, chkdirprfx+"sws.dcid.bil", dcid)  // receiving cell of downslope sws
	writeInts(gd, chkdirprfx+"sws.order.bil", sord) // computational sws ordering
}

func (ws *Subwatershed) SaveGob(fp string) error {
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
