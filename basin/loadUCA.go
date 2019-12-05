package basin

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"

	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmio"
)

func loadUCA(topo *tem.TEM, swscidxr map[int][]int, sws map[int]int, fp string) (uca map[int]map[int]int) {
	if _, ok := mmio.FileExists(fp); ok {
		fmt.Printf(" loading: %s\n", fp)
		var err error
		if uca, err = loadUCAgob(fp); err != nil {
			log.Fatalf(" loadUCA.go loadUCAgob error: %v", err)
		}
	} else {
		// compute unit contributing areas
		fmt.Print(" building uca.. ")
		type col struct {
			s int
			u map[int]int
		}
		ch := make(chan col, len(swscidxr))
		for s, cids := range swscidxr {
			go func(s int, cids []int) {
				m := make(map[int]int, len(cids))
				for _, c := range cids {
					m[c] = 1
					for _, u := range topo.UpIDs(c) {
						if sws[u] == s { // to be kept within sws
							m[c] += topo.UnitContributingArea(u)
						}
					}
				}
				ch <- col{s, m}
			}(s, cids)
		}
		// for s, cids := range swscidxr {
		// 	uca[s] = make(map[int]int, len(cids))
		// 	for _, c := range cids {
		// 		uca[s][c] = 1
		// 		for _, u := range topo.UpIDs(c) {
		// 			if sws[u] == s { // to be kept within sws
		// 				uca[s][c] += topo.UnitContributingArea(u)
		// 			}
		// 		}
		// 	}
		// }
		uca = make(map[int]map[int]int, len(swscidxr))
		for i := 0; i < len(swscidxr); i++ {
			c := <-ch
			uca[c.s] = c.u
		}
		close(ch)
		// go func() {
		fmt.Printf("saving to %s\n", fp)
		if err := saveUCAgob(uca, fp); err != nil {
			log.Fatalf(" loadUCA.go saveUCAgob error: %v", err)
		}
		// }()
	}
	return
}

func saveUCAgob(uca map[int]map[int]int, fp string) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(uca)
	if err != nil {
		return err
	}
	return nil
}

func loadUCAgob(fp string) (map[int]map[int]int, error) {
	var uca map[int]map[int]int
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&uca)
	if err != nil {
		return nil, err
	}
	return uca, nil
}
