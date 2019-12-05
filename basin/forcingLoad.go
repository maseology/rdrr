package basin

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/maseology/goHydro/met"
	"github.com/maseology/mmio"
)

// masterForcing returns forcing data from mastreDomain
func masterForcing() (*FORC, int) {
	if masterDomain.frc == nil {
		log.Fatalf(" basin.masterForcing error: masterDomain.frc == nil\n")
	}
	// if masterDomain.frc.h.Nloc() != 1 && masterDomain.frc.h.LocationCode() <= 0 {
	// 	log.Fatalf(" basin.masterForcing error: invalid *FORC type in masterDomain\n")
	// }
	if masterDomain.frc.h.LocationCode() == 0 {
		return masterDomain.frc, -1
	}
	return masterDomain.frc, int(masterDomain.frc.h.Locations[0][0].(int32))
}

// masterForcingNewOutlet reloads discharge from *.met file
func masterForcingNewOutlet(metfp string) (*FORC, int) {
	if masterDomain.frc == nil {
		log.Fatalf(" masterForcingNewOutlet error: masterDomain.frc == nil\n")
	}
	nfrc, outlet := loadForcing(metfp, true) // gauge outlet cell id found in .met file
	if nfrc == nil {
		log.Fatalf(" masterForcingNewOutlet error: %s == nil\n", metfp)
	}
	if _, ok := nfrc.h.WBDCxr()["UnitDischarge"]; !ok {
		log.Fatalf(" masterForcingNewOutlet error: no 'UnitDischarge' found in %s\n", metfp)
	}

	nintvl := int64(nfrc.h.IntervalSec())                    // time step interval [s]
	dtb, dte, intvl := masterDomain.frc.h.BeginEndInterval() // start date, end date, time step interval [s]
	if nintvl != intvl {
		log.Fatalf(" masterForcingNewOutlet TODO: unequal time steps found\n")
	}
	dnew := make([][][]float64, 3)
	dnew[0] = masterDomain.frc.c.D[0]
	dnew[1] = masterDomain.frc.c.D[1]
	dnew[2] = [][]float64{nfrc.get(dtb, dte, nfrc.h.WBDCxr()["UnitDischarge"])}

	masterDomain.frc.c.D = dnew
	masterDomain.frc.Q0 = nfrc.medQ()
	masterDomain.frc.h.SetWBDC(masterDomain.frc.h.WBCD + met.UnitDischarge)
	return masterDomain.frc, outlet
}

// LoadForcing (re-)loads forcing data
func loadForcing(fp string, print bool) (*FORC, int) {
	// import forcings
	if _, ok := mmio.FileExists(fp); !ok {
		return nil, -1
	}
	m, d, err := met.ReadMET(fp, print)
	if err != nil {
		log.Fatalln(err)
	}

	// checks
	dtb, dte, intvl := m.BeginEndInterval() // start date, end date, time step interval [s]
	temp, k := make([]temporal, m.Nstep()), 0
	x := m.WBDCxr()
	for dt := dtb; !dt.After(dte); dt = dt.Add(time.Second * time.Duration(intvl)) {
		if d.T[k] != dt {
			log.Fatalf("loadForcing error: date mis-match: %v vs %v", d.T[k], dt)
		}
		v := d.D[k][0] // [date ID][cell ID][type ID]
		// y := v[x["AtmosphericYield"]]     // precipitation/atmospheric yield (rainfall + snowmelt)
		ep := v[x["AtmosphericDemand"]] // evaporative demand
		if ep < 0. {
			d.D[k][0][x["AtmosphericDemand"]] = 0.
		}
		temp[k] = temporal{doy: dt.YearDay() - 1, mt: int(dt.Month())}
		k++
	}

	if m.Nloc() != 1 && m.LocationCode() <= 0 {
		log.Fatalf(" loadForcing error: unrecognized .met type\n")
	}
	outlet := int(m.Locations[0][0].(int32))

	return &FORC{
		c:   *d, // met.Coll
		h:   *m, // met.Header
		t:   temp,
		nam: mmio.FileName(fp, false), // station name
	}, outlet
}

func loadGOBforcing(gobdir string, print bool) (*FORC, int) {
	// import forcings
	loadGOB := func(fp string) ([][]float64, error) {
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
	loadINTSCT := func(fp string) (map[int]int, error) {
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
	var intsct map[int]int
	wg.Add(3)
	go func() {
		defer wg.Done()
		var err error
		if y, err = loadGOB(gobdir + "frc.y.gob"); err != nil {
			log.Fatalf("%v", err)
		}
		// tt.Lap(fmt.Sprintf(" %s loaded", "frc.y.gob"))
	}()
	go func() {
		defer wg.Done()
		var err error
		if ep, err = loadGOB(gobdir + "frc.ep.gob"); err != nil {
			log.Fatalf("%v", err)
		}
		// tt.Lap(fmt.Sprintf(" %s loaded", "frc.ep.gob"))
	}()
	go func() {
		defer wg.Done()
		var err error
		if intsct, err = loadINTSCT(gobdir + "metIntersect.gob"); err != nil {
			log.Fatalf("%v", err)
		}
		// tt.Lap(fmt.Sprintf(" %s loaded", "metIntersect.gob"))
	}()
	wg.Wait()
	// tt.Lap("met GOB load complete")

	var d met.Coll

	//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////// Default HARD-CODED values ///////////////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	dtb, dte, intvl := time.Date(1999, time.October, 1, 0, 0, 0, 0, time.UTC), time.Date(2019, time.September, 30, 0, 0, 0, 0, time.UTC), 86400
	h := met.NewHeader(dtb, dte, intvl, len(y), 8)
	h.SetWBDC(met.AtmosphericYield + met.AtmosphericDemand)
	if len(y[0]) != h.Nstep() {
		log.Fatalf("loadGOBforcing error: gob and date range are incompatible")
	}
	//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	temp, k := make([]temporal, h.Nstep()), 0
	d.T, d.D = make([]time.Time, h.Nstep()), make([][][]float64, 2)
	for dt := dtb; !dt.After(dte); dt = dt.Add(time.Second * time.Duration(intvl)) {
		temp[k] = temporal{doy: dt.YearDay() - 1, mt: int(dt.Month())}
		d.T[k] = dt
		d.D = [][][]float64{y, ep}
		k++
	}

	// slow
	// ncell := len(y)
	// temp, k := make([]temporal, h.Nstep()), 0
	// d.T, d.D = make([]time.Time, h.Nstep()), make([][][]float64, ncell)
	// for i := 0; i < ncell; i++ {
	// 	d.D[i] = make([][]float64, h.Nstep())
	// }
	// for dt := dtb; !dt.After(dte); dt = dt.Add(time.Second * time.Duration(intvl)) {
	// 	temp[k] = temporal{doy: dt.YearDay() - 1, mt: int(dt.Month())}
	// 	d.T[k] = dt
	// 	d.D[k] = make([][]float64, ncell)
	// 	for i := 0; i < ncell; i++ {
	// 		d.D[i][k] = []float64{y[i][k], ep[i][k]}
	// 	}
	// 	k++
	// }

	// tt.Lap("Forcing build complete")
	return &FORC{
		c:   d, // met.Coll
		h:   h, // met.Header
		t:   temp,
		x:   intsct,
		nam: "gob",
	}, -1
}
