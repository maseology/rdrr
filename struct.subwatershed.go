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

func (w *Subwatershed) checkandprint(gd *grid.Definition, cids []int, fnc float64, chkdirprfx string, crop bool) {

	var gd2 *grid.Definition
	xr := make(map[int]int)
	if crop {
		gd2, xr = gd.CropToActives()
	} else {
		gd2 = gd
		for _, c := range gd.Sactives {
			xr[c] = c
		}
	}

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

	si, sids, dsws, dcid, sds, sgw, islak := gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999), gd2.NullInt32(-9999)
	hassgw := w.Sgw != nil
	for _, c := range gd.Sactives {
		if i, ok := mx[c]; ok {
			c2 := xr[c]
			si[c2] = int32(w.Sid[i])
			sids[c2] = int32(w.Isws[w.Sid[i]])
			dsws[c2] = int32(w.Dsws[w.Sid[i]].Sid)
			dcid[c2] = int32(w.Dsws[w.Sid[i]].Cid)

			if w.Islake[w.Sid[i]] {
				islak[c2] = 1
			}
			if hassgw {
				sgw[c2] = int32(w.Sgw[w.Sid[i]])
			}
		}
	}
	for k, scids := range w.Scis {
		for i, sc := range scids {
			c := xr[cids[sc]]
			sds[c] = int32(w.Sds[k][i])
		}
	}

	sord := gd2.NullInt32(-9999)
	for k, inner := range w.Outer {
		for _, isw := range inner {
			for _, i := range wcis[isw] {
				sord[xr[cids[i]]] = int32(k + 1)
			}
		}
	}

	writeInts(gd2, chkdirprfx+"sws.aid.bil", si)   // zero-based index
	writeInts(gd2, chkdirprfx+"sws.sid.bil", sids) // original index
	writeInts(gd2, chkdirprfx+"sws.sds.bil", sds)  // cell topology per sub-watershed, <0 is routed to down-SWS
	if hassgw {
		writeInts(gd2, chkdirprfx+"sws.sgw.bil", sgw) // groundwater index, now projected to sws
	}
	writeInts(gd2, chkdirprfx+"sws.dsws.bil", dsws)    // downslope sws index
	writeInts(gd2, chkdirprfx+"sws.dcid.bil", dcid)    // receiving cell of downslope sws
	writeInts(gd2, chkdirprfx+"sws.order.bil", sord)   // computational sws ordering
	writeInts(gd2, chkdirprfx+"sws.islake.bil", islak) // shows which sws is deemed a lake
}

func (w *Subwatershed) SaveGob(fp string) error {
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf(" mapper.SaveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(w); err != nil {
		return fmt.Errorf(" mapper.SaveGob %v", err)
	}
	f.Close()
	return nil
}

func LoadGobSubwatershed(fp string) (*Subwatershed, error) {
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

func (w *Subwatershed) BuildUpSWS() map[int][]int {
	o := make(map[int][]int, len(w.Sid))
	for i := range w.Scis {
		if _, ok := o[w.Dsws[i].Sid]; !ok {
			o[w.Dsws[i].Sid] = []int{}
		}
		o[w.Dsws[i].Sid] = append(o[w.Dsws[i].Sid], i)
	}
	return o
}
