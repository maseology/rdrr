package prep

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/rdrr/basin"
)

// Cell collects cell cross-referencing
type Cell struct {
	Ki, Cid, Sid, Mid int // array index, cell ID, sws ID, meteo ID
	// PSIf              [366]float64
}

// BuildSTRC builds the structural (static) form of the model
func BuildSTRC(gobDir, gdefFP, demFP, swsFP string) (strc *basin.STRC, gd *grid.Definition, cells []Cell, sws map[int]int, nsws int) {

	var err error
	gd, err = grid.ReadGDEF(gdefFP, true)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(gd.Sactives) <= 0 {
		log.Fatalf("error: grid definition requires active cells")
	}

	var dem tem.TEM
	if err = dem.New(demFP); err != nil {
		log.Fatalf(" tem.New() error: %v", err)
	}
	for _, i := range gd.Sactives {
		if dem.TEC[i].Z == -9999. {
			// log.Fatalf("no elevation assigned to cell %d", i)
			fmt.Printf(" WARNING no elevation assigned to meteo cell %d\n", i)
		}
	}

	strc = &basin.STRC{
		TEM:   &dem,
		UpCnt: dem.ContributingCellMap(),
		Acell: gd.Cw * gd.Cw,
		Wcell: gd.Cw,
	}

	fmt.Println(" collecting SWSs..")
	sws, nsws = func() (map[int]int, int) {
		var gsws grid.Indx
		gsws.LoadGDef(gd)
		gsws.New(swsFP, false)
		return gsws.Values(), len(gsws.UniqueValues())
	}()

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

	cells = make([]Cell, gd.Na)
	for k := 0; k < gd.Na; k++ {
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

	if err = strc.SaveGob(gobDir + "STRC.gob"); err != nil {
		log.Fatalf(" BuildSTRC error: %v", err)
	}

	return

}
