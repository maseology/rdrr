package rdrr

import "sort"

// the following simplified the gw zones by assigning only one per sws
func (w *Subwatershed) remapGWzones(mp *Mapper) (fngwcnew []float64, igwnew []int) {

	m := make(map[int]map[int][]int, w.Ns)
	for is, cids := range w.Scis {
		m[is] = make(map[int][]int)
		for _, cid := range cids {
			m[is][mp.Igw[cid]] = append(m[is][mp.Igw[cid]], cid)
		}
	}

	agwnew := make([]int, w.Ns)
	fngwcnew = make([]float64, len(mp.Fngwc))
	for is, s := range m {
		igs, ns := 0, 0
		for ig, cids := range s {
			n := len(cids)
			if n > ns {
				ns = n
				igs = ig
			}
		}
		agwnew[is] = igs
		fngwcnew[igs] += float64(len(w.Scis[is]))
	}

	igwnew = make([]int, len(mp.Igw))
	for i, si := range w.Sid {
		igwnew[i] = agwnew[si]
	}

	// trim unused gwres
	func() {
		a := []int{}
		for i, n := range fngwcnew {
			if n <= 0 {
				a = append(a, i)
			}
		}
		if len(a) == 0 {
			return
		}
		sort.Ints(a)
		for i := len(a) - 1; i >= 0; i-- {
			rem := a[i]
			for j := 0; j < len(agwnew); j++ {
				if agwnew[j] > rem {
					agwnew[j]--
				}
			}
			for j := 0; j < len(igwnew); j++ {
				if igwnew[j] > rem {
					igwnew[j]--
				}
			}
			fngwcnew = append(fngwcnew[:rem], fngwcnew[rem+1:]...)
		}
	}()

	w.Sgw = agwnew
	return fngwcnew, igwnew
}

// func() { // remove gw reservoirs not in model domain
// 	remove := func(slice [][]int, s int) [][]int {
// 		return append(slice[:s], slice[s+1:]...)
// 	}
// 	s := []int{}
// 	for i, a := range gcx {
// 		if len(a) == 0 {
// 			s = append(s, i)
// 		}
// 	}
// 	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
// 		s[i], s[j] = s[j], s[i]
// 	}
// 	for _, i := range s {
// 		gcx = remove(gcx, i)
// 	}
// }()
