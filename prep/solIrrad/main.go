package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

func main() {
	sif := openSif("S:/ormgp_rdrr/ORMGP_50_hydrocorrect.uhdem.sif.gob")
	fmt.Println(&sif)
}

func openSif(fp string) map[int][366]float64 {
	var sif map[int][366]float64
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&sif)
	if err != nil {
		log.Fatal(err)
	}
	return sif
}
