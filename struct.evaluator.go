package rdrr

import "github.com/maseology/goHydro/convolution"

type Evaluator struct {
	Schannel                             []*convolution.Convolution
	Outer, Sais, Sads, Smon              [][]int
	Drel, Bo, Fcasc, Finf, DepSto, Efact [][]float64
	IsStrm                               [][]bool
	Sgw, Dsws                            []int
	M, Fngwc                             []float64
	Nc, Nm                               int
	// Eafact, Dext                  float64 // open water evaporation coefficient, extinction depth
	// IsLake                        []bool
}
