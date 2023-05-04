package rdrr

func (sws *Subwatershed) BuildSwsTransfers() ([][]int, [][]int) {
	ns := len(sws.Scis)
	dwnas := make([][]int, ns) // [down-slope sws id; receiving array index]
	m := make(map[int][]int)
	for _, v := range sws.Dsws {
		m[v[0]] = append(m[v[0]], v[1])
	}
	incs := make([][]int, ns) // down-sws receiver cells indexing xsv
	for is := range sws.Dsws {
		if n, ok := m[is]; ok {
			incs[is] = make([]int, len(n))
			for i, c := range n {
				incs[is][i] = c
			}
		}
	}

	indexof := func(s, v int) int {
		for i, c := range incs[s] {
			if c == v {
				return i
			}
		}
		panic("Evaluate incs error")
	}
	for is, ia := range sws.Dsws {
		if ia[0] > -1 {
			dwnas[is] = []int{ia[0], indexof(ia[0], ia[1])} // [down-slope sws id; receiving array index]
		}
	}
	return dwnas, incs
}
