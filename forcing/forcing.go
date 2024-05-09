package forcing

import "time"

type Forcing struct {
	T           []time.Time // [date ID]
	Ya, Ea      [][]float64 // [staID][DateID] atmospheric exchange terms
	IntervalSec float64
}
