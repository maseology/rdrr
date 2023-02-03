package rdrr

import (
	"fmt"
	"sort"

	"github.com/maseology/goHydro/grid"
)

func (s *Structure) loadSWS(swsfp string) Subwatershed {

	sids := func(fp string) []int {
		fmt.Printf(" loading: %s\n", fp)
		var g grid.Indx
		g.LoadGDef(s.GD)
		g.NewShort(fp, true)
		m := g.Values()
		aout := make([]int, s.Nc)
		for i, c := range s.Cids {
			if v, ok := m[c]; ok {
				aout[i] = v
			} else {
				panic("loadIndx error: " + fp)
			}
		}
		return aout
	}(swsfp)

	// set mapped sws IDs to a 0-base array index, sorted on input zone ID
	xsws, isws := func() (map[int]int, []int) {
		d := make(map[int]int)
		for i := range s.Cids {
			d[sids[i]]++
		}
		u := make([]int, 0, len(d))
		for k := range d {
			u = append(u, k)
		}
		sort.Ints(u)
		for i, uu := range u {
			if _, ok := d[uu]; !ok {
				panic("xgw error 1")
			}
			d[uu] = i
		}
		return d, u
	}()

	fnsc := make([]float64, len(xsws))
	for i := range s.Cids {
		if isw, ok := xsws[sids[i]]; ok {
			sids[i] = isw // reset mapped gw zone IDs to a 0-base array index
			fnsc[isw]++
		} else {
			panic("loadSWS isws error")
		}
	}

	mcids := make(map[int][]int, len(xsws))
	for i := range s.Cids { // topo-safe cell order
		mcids[sids[i]] = append(mcids[sids[i]], i)
	}
	scids := make([][]int, len(mcids))
	sds := make([][]int, len(mcids))
	newds := func(sid int, scids []int) []int {
		m := make(map[int]int, len(scids))
		dsc := make([]int, len(scids))
		for i, c := range scids {
			dsc[i] = s.Ds[c]
			m[c] = i
		}

		ds := make([]int, len(dsc))
		for i, k := range dsc {
			if ids, ok := m[k]; ok {
				ds[i] = ids
			} else {
				ds[i] = -1
			}
		}

		return ds
	}
	for k, v := range mcids {
		sds[k] = newds(k, v)
		scids[k] = v
	}

	dsws := func() [][]int {
		dsws := make([][]int, len(scids))
		for is, c := range scids {
			oi := len(c) - 1
			if sds[is][oi] != -1 {
				panic("loadSWS wtf")
			}
			di := s.Ds[c[oi]]
			if di > -1 {
				ds := sids[di]
				dc := func() int {
					dsc := scids[ds]
					for j := len(dsc) - 1; j >= 0; j-- {
						if dsc[j] == di {
							return j
						}
					}
					return -1
				}()
				if dsws[is] != nil {
					panic("loadSWS expecting only 1 outlet per sws")
				}
				dsws[is] = []int{ds, dc}
			} else {
				dsws[is] = []int{-1, -1} // model outlet
			}
		}

		return dsws
	}()

	return Subwatershed{
		Scis: scids,     // set of cell indices per sws
		Sid:  sids,      // cell index to 0-based sws index
		Sds:  sds,       // new cell topology per sub-watershed
		Dsws: dsws,      // [downslope sub-watershed,cell index receiving input], -1 out of model
		Isws: isws,      // sws index to sub-watersed ID (needed for forcings)
		Fnsc: fnsc,      // number of cells per sws
		Ns:   len(xsws), // number od sws's
	}
}
