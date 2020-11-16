package main

import (
	"encoding/gob"
	"fmt"
	"os"
)

func main() {
	d, _ := loadGOB("M:/OWRC-RDRR/met/frc.ys.gob")
	fmt.Println(len(d), len(d[0]))
	for i, a := range d {
		s := 0.
		for _, v := range a {
			s += v
		}
		fmt.Println(i, s/float64(len(a))*4.*365.24*1000.)
	}
}

func loadGOB(fp string) ([][]float64, error) {
	var d [][]float64
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&d)
	if err != nil {
		return nil, err
	}
	return d, nil
}
