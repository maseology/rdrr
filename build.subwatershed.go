package rdrr

import (
	"fmt"
	"log"
	"sort"

	"github.com/maseology/goHydro/grid"
)

func (s *Structure) loadSWS(swsfp string) Subwatershed {

	// getting sws IDs mapped to topo-safe array (note asids is altered below once sws IDs are remapped to a 0-based array)
	asids := func(fp string) []int {
		fmt.Printf("   loading: %s\n", fp)
		var g grid.Indx
		g.GD = s.GD
		g.New(fp) //, true)
		aout := make([]int, s.Nc)
		nrm := 0
		for i, c := range s.Cids { // topo-safe cell order
			if v, ok := g.A[c]; ok {
				if v < 0 {
					nrm++
					continue
				}
				aout[i] = v
			} else {
				panic("loadSWS loadIndx error: " + fp)
			}
		}
		if nrm > 0 {
			log.Fatalf("    ERROR: %d cells (%.3f%%) were not assigned a positive SWS ID.\n     Likely due to the trimming of small SWSs.\n     Re-assign GDEF to the SWS layer to avoid this message.", nrm, float64(nrm)*100./float64(s.Nc))
		}
		return aout
	}(swsfp)

	// mapping sorted sws IDs to a 0-based array index
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
				panic("loadSWS xsws error")
			}
			d[uu] = i
		}
		return d, u
	}()

	// checking for consistency, remapping asids to the 0-based sws id, and gathering sws sizes
	fnsc := make([]float64, len(xsws))
	for i := range s.Cids {
		if isw, ok := xsws[asids[i]]; ok {
			asids[i] = isw // reset mapped sws IDs to a 0-based array index
			fnsc[isw]++
		} else {
			panic("loadSWS isws error")
		}
	}

	// collecting lists of acids per aswsids
	mcids := make(map[int][]int, len(xsws))
	for i := range s.Cids { // topo-safe cell order
		mcids[asids[i]] = append(mcids[asids[i]], i) // topo-safe cell order on a per-sws basis
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
				ds[i] = -1 // not draining to cell within current sws, setting to <0 will force rdrr to drain to downslope SWS
			}
		}
		return ds
	}

	// remapping mcids to a list of lists, building downslopes on a per-sws basis
	scids := make([][]int, len(mcids))
	sds := make([][]int, len(mcids))
	for k, v := range mcids {
		sds[k] = newds(v)
		scids[k] = v
	}

	// collecting sws topologies
	dsws := func() []SWStopo {
		dsws := make([]SWStopo, len(scids))
		for is, c := range scids {
			oi := len(c) - 1 // final cell id drains to downslope SWS
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
		Scis:   scids, // set of cell indices per sws
		Sds:    sds,   // cell topology per sub-watershed, <0 is routed to down-SWS
		Dsws:   dsws,  // [downslope sub-watershed,cell index receiving input], -1 out of model
		Sid:    asids, // 0-based cell index to 0-based sws index
		Isws:   isws,  // sws index to sub-watersed ID (needed for forcings)
		Fnsc:   fnsc,  // number of cells per sws
		Islake: make([]bool, len(xsws)),
		Ns:     len(xsws), // number of sws's
	}
}

func (w *Subwatershed) UpdateDS(s *Structure) {
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

	mcids := make(map[int][]int, w.Ns)
	for i := range s.Cids { // topo-safe cell order
		mcids[w.Sid[i]] = append(mcids[w.Sid[i]], i)
	}
	scids := make([][]int, len(mcids))
	sds := make([][]int, len(mcids))
	for k, v := range mcids {
		sds[k] = newds(v)
		scids[k] = v
	}
	w.Sds = sds
	w.Scis = scids
}
