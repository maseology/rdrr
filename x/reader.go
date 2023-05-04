package rdrr

import (
	"encoding/gob"
	"log"
	"os"
)

// a set of hashable time-values (hashing with time.Time is not desirable)
type hyd map[int64]float64

func saveHyd(fp string, h hyd) {
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(h)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
}

func loadHyd(fp string) hyd {
	var d hyd
	f, err := os.Open(fp)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	enc := gob.NewDecoder(f)
	err = enc.Decode(&d)
	if err != nil {
		log.Fatal(err)
	}
	return d
}

// type hyd struct {
// 	Date time.Time
// 	Val  float64
// }

// func saveHyd(fp string, h []hyd) {
// 	f, err := os.Create(fp)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	enc := gob.NewEncoder(f)
// 	err = enc.Encode(h)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	f.Close()
// }

// func loadHyd(fp string) []hyd {
// 	var d []hyd
// 	f, err := os.Open(fp)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer f.Close()
// 	enc := gob.NewDecoder(f)
// 	err = enc.Decode(&d)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	return d
// }
