package model

import (
	"sort"
)

// Prints a set of rasters for verification
func PrintInts(fp string, cids []int, vals map[int]int) {
	ocids := func() []int {
		ocids := make([]int, len(cids))
		copy(ocids, cids)
		sort.Ints(ocids)
		return ocids
	}()

	a := make([]int32, len(cids))
	for i, cid := range ocids {
		if vv, ok := vals[cid]; ok {
			a[i] = int32(vv)
			continue
		}
		panic("error with " + fp)
	}
	writeInts(fp, a)
}
