package rdrr

func (s *Structure) buildStreams() ([]int, int) {
	strmcthresh := int(strmkm2 * 1000. * 1000. / s.GD.CellArea()) // "stream cell" threshold
	strms, nstrm := []int{}, 0
	for i := range s.Cids {
		if s.Upcnt[i] > strmcthresh {
			strms = append(strms, i)
			nstrm++
		}
	}
	o := make([]int, nstrm)
	copy(o, strms)
	return o, nstrm
}
