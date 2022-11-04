package model

import (
	"sort"

	"github.com/maseology/mmio"
)

// Prints a set of rasters for verification
func (s *STRC) PrintAndCheck(dir string) []int {
	ocids := make([]int, len(s.CIDs))
	copy(ocids, s.CIDs)
	sort.Ints(ocids)

	uc := make([]int32, len(s.CIDs))
	dg := make([]float64, len(s.CIDs))
	for i, c := range ocids {
		uc[i] = int32(s.UpCnt[c])
		dg[i] = s.DwnGrad[c]
	}
	writeInts(dir+"/check/STRC-upcounts.bin", uc)
	writeFloats(dir+"/check/STRC-dwngrad.bin", dg)
	mmio.WriteFloats(dir+"/check/STRC-dwngrad.txt", dg)

	return ocids
}
