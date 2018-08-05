package main

import (
	"fmt"

	"github.com/maseology/rdrr/gwru"

	"github.com/maseology/rdrr/tem"
)

func main() {

	// load topography
	var t tem.TEM
	t.New("C:/Users/mason/go/src/test/rdrr_test/input/ORMGP_50_hydrocorrect_carruthers.uhdem")

	// load surfical material ksat
	ksat := make(map[int]float64)
	for i := range t.TECs {
		ksat[i] = 0.0001
	}

	// build Topodel
	var g gwru.TOPMODEL
	g.New(ksat, t, 50., .1, 5., .1)

	i, s := 0, 0.
	for _, v := range g.Di {
		fmt.Println(i, v)
		i++
		s += v
	}
	s /= float64(i)
	fmt.Println(i, s)
}
