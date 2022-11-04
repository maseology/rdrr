package model

import "fmt"

func (frc *FORC) PrintAndCheck(dir string, ocids []int) error {

	mid := make([]int32, len(frc.XR))
	for i, c := range ocids {
		mid[i] = int32(frc.XR[c])
	}
	writeInts(dir+"/check/FORC-mid.indx", mid)

	ysum, esum := 0., 0.
	for j := range frc.T {
		for i := 0; i < len(frc.Ya); i++ {
			ysum += frc.Ya[i][j]
			esum += frc.Ea[i][j]
		}
	}
	ysum *= 1. / float64(len(frc.T)) / float64(len(frc.Ya)) * 4 * 365.24
	esum *= 1. / float64(len(frc.T)) / float64(len(frc.Ya)) * 4 * 365.24
	fmt.Printf(" average annual [m/yr]:\n    yeild =  %.4f\n   demand =  %.4f\n", ysum, esum)

	return nil
}
