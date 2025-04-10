package rdrr

import "github.com/maseology/goHydro/convolution"

type Evaluator struct {
	Schannel                      []*convolution.Convolution
	Outer, Sais, Sads             [][]int
	Drel, Bo, Fcasc, Finf, DepSto [][]float64
	IsStrm                        [][]bool
	Sgw, Dsws, Smon               []int
	M, Fngwc                      []float64
	Eafact, Dext                  float64 // open water evaporation coefficient, extinction depth
	Nc, Nm                        int
	// IsLake                        []bool
}
