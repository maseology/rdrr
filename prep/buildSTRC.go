package prep

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

// Cell collects cell cross-referencing
type Cell struct {
	Ki, Cid, Sid, Mid int // array index, cell ID, sws ID, meteo ID
	// PSIf              [366]float64
}

// BuildSTRC builds the structural (static) form of the model
func BuildSTRC(gd *grid.Definition, sws map[int]int, gobDir, demFP string) (strc *model.STRC, cells []Cell) {

	dem := func() tem.TEM {
		if mmio.GetExtension(demFP) == ".gob" {
			t, err := tem.LoadGob(demFP)
			if err != nil {
				log.Fatalf(" BuildSTRC tem.LoadGob() error: %v", err)
			}
			return t
		}

		var dem tem.TEM
		if err := dem.New(demFP); err != nil {
			log.Fatalf(" BuildSTRC tem.New() error: %v", err)
		}
		for _, i := range gd.Sactives {
			if _, ok := dem.TEC[i]; !ok {
				log.Fatalf(" BuildSTRC error, cell id %d not found in %s", i, demFP)
			}
			if dem.TEC[i].Z == -9999. {
				// log.Fatalf("no elevation assigned to cell %d", i)
				fmt.Printf(" WARNING no elevation assigned to cell %d\n", i)
			}
		}
		if gd.Nact != len(dem.TEC) {
			log.Fatalf("BuildSTRC todo1")
			// d := make(map[int]tem.TEC, gd.Nact)
			// for _, i := range gd.Sactives {
			// 	d[i] = dem.TEC[i]
			// 	if !gd.IsActive(d[i].Ds) {
			// 		t := d[i]
			// 		t.Ds = -1
			// 		d[i] = t
			// 	}
			// }
			// dem.TEC = d
			// dem.BuildUpslopes()
		}
		return dem
	}()

	strc = &model.STRC{
		TEM:   &dem,
		UpCnt: dem.ContributingCellMap(),
		Acell: gd.Cwidth * gd.Cwidth,
		Wcell: gd.Cwidth,
	}

	fmt.Println(" building cell geometry (skipping solirrad calculations)..")
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
					// latitude, _, err := UTM.ToLatLon(s.x, s.y, 17, "", true)
					// if err != nil {
					// 	fmt.Println(s)
					// 	log.Fatalf(" newGeomStream error: %v -- (x,y)=(%f, %f); cid: %d\n", err, s.x, s.y, s.cid)
					// }
					// si := solirrad.New(latitude, math.Tan(s.t.G), math.Pi/2.-s.t.A)
					// outputStream <- Cell{Ki: s.k, Cid: s.cid, Sid: sws[s.cid], Mid: sws[s.cid], PSIf: si.PSIfactor}
					outputStream <- Cell{Ki: s.k, Cid: s.cid, Sid: sws[s.cid], Mid: sws[s.cid]}
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

	cells = make([]Cell, gd.Nact)

	for k := 0; k < gd.Nact; k++ {
		c := <-outputStream
		cells[c.Ki] = c
	}
	close(done)
	wg.Wait()
	// close(inputStream)
	// close(outputStream)

	// func() error {
	// 	f, err := os.Create(demFP + ".Cells.gob")
	// 	defer f.Close()
	// 	if err != nil {
	// 		log.Fatalf(" cells to gob error: %v", err)
	// 	}
	// 	enc := gob.NewEncoder(f)
	// 	err = enc.Encode(cells)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return nil
	// }()

	if err := strc.SaveGob(gobDir + "STRC.gob"); err != nil {
		log.Fatalf(" BuildSTRC error: %v", err)
	}

	return

}
