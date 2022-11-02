package model

import "fmt"

// // FORC holds forcing data
// type FORC struct {
// 	T      []time.Time // [date ID]
// 	Ya, Ea [][]float64 // [staID][DateID] atmospheric exchange terms
// 	// O           [][]float64 // observed discharge (use Oxr for cross-reference)
// 	XR map[int]int // mapping of model grid cell id to met index
// 	// Oxr, mt     []int       // mapping of outlet cell ID to O[][]
// 	mt          []int // month [1,12] cross-reference
// 	IntervalSec float64
// }

// SaveGob FORC to gob
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
