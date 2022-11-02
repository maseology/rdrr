package prep

import "rdrr2/model"

const strmkm2 = 1. // total drainage area [km²] required to deem a cell a "stream cell"

func buildStreams(strc *model.STRC, cids []int) ([]int, int) {
	strmcthresh := int(strmkm2 * 1000. * 1000. / strc.Wcell / strc.Wcell) // "stream cell" threshold
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
