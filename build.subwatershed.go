package rdrr

import (
	"fmt"
	"sort"

	"github.com/maseology/goHydro/grid"
)

func (s *Structure) loadSWS(swsfp string) Subwatershed {

	asids := func(fp string) []int {
		fmt.Printf("   loading: %s\n", fp)
		var g grid.Indx
		g.GD = s.GD
		g.New(fp) //, true)
		aout := make([]int, s.Nc)
		for i, c := range s.Cids { // topo-safe cell order
			if v, ok := g.A[c]; ok {
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
			d[asids[i]]++
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
		if isw, ok := xsws[asids[i]]; ok {
			asids[i] = isw // reset mapped gw zone IDs to a 0-base array index
			fnsc[isw]++
		} else {
			panic("loadSWS isws error")
		}
	}

	newds := func(scids []int) []int {
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

	mcids := make(map[int][]int, len(xsws))
	for i := range s.Cids { // topo-safe cell order
		mcids[asids[i]] = append(mcids[asids[i]], i)
	}
	scids := make([][]int, len(mcids))
	sds := make([][]int, len(mcids))
	for k, v := range mcids {
		sds[k] = newds(v)
		scids[k] = v
	}

	dsws := func() []SWStopo {
		dsws := make([]SWStopo, len(scids))
		for is, c := range scids {
			oi := len(c) - 1
			if sds[is][oi] != -1 {
				panic("loadSWS SWStopo err")
			}
			di := s.Ds[c[oi]]
			if di > -1 {
				ds := asids[di]
				dc := func() int {
					dsc := scids[ds]
					for j := len(dsc) - 1; j >= 0; j-- {
						if dsc[j] == di {
							return j
						}
					}
					return -1
				}()
				dsws[is] = SWStopo{ds, dc}
			} else {
				dsws[is] = SWStopo{-1, -1} // model outlet
			}
		}
		return dsws
	}()

	return Subwatershed{
		Scis: scids,     // set of cell indices per sws
		Sds:  sds,       // cell topology per sub-watershed
		Dsws: dsws,      // [downslope sub-watershed,cell index receiving input], -1 out of model
		Sid:  asids,     // 0-based cell index to 0-based sws index
		Isws: isws,      // sws index to sub-watersed ID (needed for forcings)
		Fnsc: fnsc,      // number of cells per sws
		Ns:   len(xsws), // number of sws's
	}
}
