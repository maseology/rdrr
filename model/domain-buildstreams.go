package model

// BuildStreams determines stream cells based on const strmkm2
func BuildStreams(strc *STRC, cids []int) ([]int, int) {
	strmcthresh := int(strmkm2 * 1000. * 1000. / strc.Acell) // "stream cell" threshold
	strms, nstrm := []int{}, 0
	for _, c := range cids {
		if strc.UpCnt[c] > strmcthresh {
			strms = append(strms, c)
			nstrm++
		}
	}
	o := make([]int, nstrm)
	copy(o, strms)
	return o, nstrm
}
