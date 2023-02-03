package rdrr

import "sort"

func (s *Structure) buildGWzone(gids []int) (fngwc []float64, igw []int) {
	if gids == nil {
		gids = make([]int, s.Nc)
		for i := range s.Cids {
			gids[i] = 0 // defaulting to complete gw zone
		}
		fngwc = []float64{float64(s.Nc)}
	} else {
		if len(gids) != s.Nc {
			panic("buildGWzone count error")
		}

		// set mapped gw zone IDs to a 0-base array index, sorted on input zone ID
		xgw := func() map[int]int {
			d := make(map[int]int)
			for i := range s.Cids {
				d[gids[i]]++
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
			return d
		}()
		_ = xgw

		fngwc = make([]float64, len(xgw))
		for i := range s.Cids {
			if ig, ok := xgw[gids[i]]; ok {
				gids[i] = ig // reset mapped gw zone IDs to a 0-base array index
				fngwc[ig]++
			} else {
				panic("buildGWzone ig error")
			}
		}
	}

	return fngwc, gids
}
