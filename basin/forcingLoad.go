package basin

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

func loadGOBforcing(gobdir string) (*FORC, int) {
	// import forcings
	loadData := func(fp string) ([][]float64, error) {
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
	loadDT := func(fp string) ([]time.Time, error) {
		var d []time.Time
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
	loadXR := func(fp string) (map[int]int, error) {
		var d map[int]int
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

	// tt := mmio.NewTimer()
	var wg sync.WaitGroup
	fmt.Printf(" loading met GOBs from %s\n", gobdir)
	var y, ep [][]float64
	var mxr map[int]int
	var dts []time.Time
	wg.Add(4)
	go func() {
		defer wg.Done()
		var err error
		if y, err = loadData(gobdir + "frc.y.gob"); err != nil {
			log.Fatalf("%v", err)
		}
		// tt.Lap(fmt.Sprintf(" %s loaded", "frc.y.gob"))
	}()
	go func() {
		defer wg.Done()
		var err error
		if ep, err = loadData(gobdir + "frc.ep.gob"); err != nil {
			log.Fatalf("%v", err)
		}
		// tt.Lap(fmt.Sprintf(" %s loaded", "frc.ep.gob"))
	}()
	go func() {
		defer wg.Done()
		var err error
		if mxr, err = loadXR(gobdir + "frc.xr.gob"); err != nil {
			log.Fatalf("%v", err)
		}
		// tt.Lap(fmt.Sprintf(" %s loaded", "frc.xr.gob"))
	}()
	go func() {
		defer wg.Done()
		var err error
		if dts, err = loadDT(gobdir + "frc.dts.gob"); err != nil {
			log.Fatalf("%v", err)
		}
		// tt.Lap(fmt.Sprintf(" %s loaded", "frc.dts.gob"))
	}()
	wg.Wait()
	// tt.Lap("met GOB load complete")

	// dtb, dte, intvl := time.Date(1989, time.October, 1, 0, 0, 0, 0, time.UTC), time.Date(2019, time.September, 30, 0, 0, 0, 0, time.UTC), 86400
	nstp := len(dts)
	dtb, dte, intvl := dts[0], dts[nstp-1], int(dts[1].Sub(dts[0]).Seconds())
	if intvl != 86400/4 {
		log.Fatalf(" intvl error, %d", intvl)
	}

	t, d, mt, k := make([]time.Time, nstp), make([][][]float64, 2), make([]int, nstp), 0
	for dt := dtb; !dt.After(dte); dt = dt.Add(time.Second * time.Duration(intvl)) {
		t[k] = dt
		mt[k] = int(dt.Month())
		d = [][][]float64{y, ep}
		k++
	}

	// tt.Lap("Forcing build complete")
	return &FORC{
		T:  t,
		D:  d,
		XR: mxr,
		mt: mt,
		// nam: "gob",
	}, -1
}
