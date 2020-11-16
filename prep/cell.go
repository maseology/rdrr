package prep

import (
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"os"
	"sync"

	"github.com/im7mortal/UTM"
	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/solirrad"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmio"
)

type Cell struct {
	K, Cid, SwsID int
	PSIf          [366]float64
}

func GetCells(gdefFP, demFP, swsFP string) ([]Cell, *grid.Definition, int) {
	gd, err := grid.ReadGDEF(gdefFP, true)
	if err != nil {
		log.Fatalf("%v", err)
	}
	nact := len(gd.Sactives)
	if nact <= 0 {
		log.Fatalf("error: grid definition requires active cells")
	}

	var dem tem.TEM
	if _, ok := mmio.FileExists(demFP + ".TEM.gob"); ok {
		var err error
		fmt.Println(" loading TEM from gob..")
		dem, err = tem.LoadGob(demFP + ".TEM.gob")
		if err != nil {
			log.Fatalf(" tem gob read error: %v", err)
		}
	} else {
		if err := dem.New(demFP); err != nil {
			log.Fatalf(" tem.New() error: %v", err)
		}
		if err := dem.SaveGob(demFP + ".TEM.gob"); err != nil {
			log.Fatalf(" tem.Save() error: %v", err)
		}
	}
	for _, i := range gd.Sactives {
		if dem.TEC[i].Z == -9999. {
			// log.Fatalf("no elevation assigned to cell %d", i)
			fmt.Printf(" WARNING no elevation assigned to meteo cell %d\n", i)
		}
	}

	if _, ok := mmio.FileExists(demFP + ".ContributingCellMap.gob"); !ok {
		fmt.Println(" building contributing cell map gob..")
		ucnt := dem.ContributingCellMap()
		if err := mmio.SaveGOB(demFP+".ContributingCellMap.gob", ucnt); err != nil {
			log.Fatalf(" topo.ContributingCellMap error: %v", err)
		}
	}

	fmt.Println(" +++ MM: SHOULD BE SPAWNING SOME TEM GOBS HERE +++++++++++++++++++")

	fmt.Println(" collecting SWSs..")
	var gsws grid.Indx
	gsws.LoadGDef(gd)
	gsws.New(swsFP, false)
	sws := gsws.Values()

	// var uca map[int]map[int]int // unit contributing areas
	// readUCA := func() {
	// 	tt := mmio.NewTimer()
	// 	// defer wg.Done()
	// 	fp := mmio.RemoveExtension(swsFP) + ".uca.gob"
	// 	// fmt.Printf(" loading: %s\n", fp)
	// 	uca = func(topo *tem.TEM, swscidxr map[int][]int, sws map[int]int, fp string) (uca map[int]map[int]int) {
	// 		if _, ok := mmio.FileExists(fp); ok {
	// 			fmt.Printf(" loading: %s\n", fp)
	// 			var err error
	// 			if uca, err = loadUCAgob(fp); err != nil {
	// 				log.Fatalf(" loadUCA.go loadUCAgob error: %v", err)
	// 			}
	// 		} else {
	// 			// compute unit contributing areas
	// 			fmt.Print(" building uca.. ")
	// 			type col struct {
	// 				s int
	// 				u map[int]int
	// 			}
	// 			ch := make(chan col, len(swscidxr))
	// 			for s, cids := range swscidxr {
	// 				go func(s int, cids []int) {
	// 					m := make(map[int]int, len(cids))
	// 					for _, c := range cids {
	// 						m[c] = 1
	// 						for _, u := range topo.UpIDs(c) {
	// 							if sws[u] == s { // to be kept within sws
	// 								m[c] += topo.UnitContributingArea(u)
	// 							}
	// 						}
	// 					}
	// 					ch <- col{s, m}
	// 				}(s, cids)
	// 			}
	// 			uca = make(map[int]map[int]int, len(swscidxr))
	// 			for i := 0; i < len(swscidxr); i++ {
	// 				c := <-ch
	// 				uca[c.s] = c.u
	// 			}
	// 			close(ch)
	// 			// go func() {
	// 			fmt.Printf("saving to %s\n", fp)
	// 			if err := saveUCAgob(uca, fp); err != nil {
	// 				log.Fatalf(" loadUCA.go saveUCAgob error: %v", err)
	// 			}
	// 		}
	// 		return
	// 	}(&dem, swscidxr, sws, fp)
	// 	tt.Lap("UCA loaded")
	// }
	// readUCA()

	var cells []Cell
	if _, ok := mmio.FileExists(demFP + ".Cells.gob"); ok {
		log.Fatalf("TODO")
	} else {
		fmt.Println(" building cell solar geometry..")
		type in1 struct {
			t      tem.TEC
			k, cid int
			x, y   float64
		}
		generateInput := func(inputStream chan<- in1) {
			for k, cid := range gd.Sactives {
				xy := gd.Coord[cid]
				inputStream <- in1{dem.TEC[cid], k, cid, xy.X, xy.Y}
			}
		}

		newStreamer := func(wg *sync.WaitGroup, done <-chan interface{}, inputStream <-chan in1, outputStream chan<- Cell) {
			defer wg.Done()
			go func() {
				for {
					select {
					case s := <-inputStream:
						latitude, _, err := UTM.ToLatLon(s.x, s.y, 17, "", true)
						if err != nil {
							fmt.Println(s)
							log.Fatalf(" newGeomStream error: %v -- (x,y)=(%f, %f); cid: %d\n", err, s.x, s.y, s.cid)
						}
						si := solirrad.New(latitude, math.Tan(s.t.G), math.Pi/2.-s.t.A)
						outputStream <- Cell{K: s.k, Cid: s.cid, SwsID: sws[s.cid], PSIf: si.PSIfactor}
					case <-done:
						return
					}
				}
			}()
		}

		done := make(chan interface{})
		inputStream := make(chan in1)
		outputStream := make(chan Cell)
		var wg sync.WaitGroup
		wg.Add(64)
		for k := 0; k < 64; k++ {
			newStreamer(&wg, done, inputStream, outputStream)
		}
		go generateInput(inputStream)

		cells = make([]Cell, nact)
		for k := 0; k < nact; k++ {
			c := <-outputStream
			cells[c.K] = c
		}
		close(done)
		wg.Wait()
		// close(inputStream)
		// close(outputStream)

		func() error {
			f, err := os.Create(demFP + ".Cells.gob")
			defer f.Close()
			if err != nil {
				log.Fatalf(" cells to gob error: %v", err)
			}
			enc := gob.NewEncoder(f)
			err = enc.Encode(cells)
			if err != nil {
				return err
			}
			return nil
		}()
	}

	return cells, gd, nact
}
