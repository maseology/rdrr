package rdrr

import "github.com/maseology/mmaths/slice"

func (w *Subwatershed) buildComputationalOrder1() {
	w.Outer = func() [][]int { // topo-safe [order, swsid]; [swsid]cids (all zero-based)

		var recurs func(i, l int)
		cnt := make(map[int]int, w.Ns)
		recurs = func(i, l int) {
			if l >= cnt[i] {
				cnt[i] = l + 1
			}
			if dsi := w.Dsws[i].Sid; dsi > -1 {
				recurs(dsi, cnt[i])
			}
		}

		for i := range w.Isws {
			recurs(i, cnt[i])
		}

		mord, lord := slice.InvertMap(cnt)
		ord := make([][]int, len(lord)) // concurrent-safe ordering of subwatersheds

		for i, k := range lord {
			cpy := make([]int, len(mord[k]))
			copy(cpy, mord[k])
			ord[i] = cpy
		}

		return ord
	}()
}
