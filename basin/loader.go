package basin

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/lusg"
)

// Loader holds the required input filepaths
type Loader struct{ Dir, Fmet, Fgd, Fhdem, Fsws, Flu, Fsg, Fobs string }

func (l *Loader) load(buildEp bool) (*FORC, STRC, MAPR, RTR, *grid.Definition, []int) {
	var wg sync.WaitGroup

	// import forcings
	var frc *FORC
	readmet := func() {
		defer wg.Done()
		if len(l.Fmet) > 0 {
			tt := mmio.NewTimer()
			fmt.Printf(" loading: %s\n", l.Fmet)
			frc, _ = loadForcing(l.Fmet, true)
			tt.Lap("met loaded")
		} else {
			frc = nil
		}
	}

	// import structural data and mapping arrays
	gd, err := grid.ReadGDEF(l.Fgd)
	if err != nil {
		log.Fatalf(" grid.ReadGDEF: %v", err)
	}
	var t tem.TEM
	var lu lusg.LandUseColl
	var sg lusg.SurfGeoColl
	var ilu, isg, sws, dsws, ucnt map[int]int
	var swscidxr map[int][]int

	wg.Add(1)
	go readmet()

	readtopo := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.Fhdem)
		if _, ok := mmio.FileExists(l.Fhdem + ".TEM.gob"); ok {
			var err error
			t, err = tem.Load(l.Fhdem + ".TEM.gob")
			if err != nil {
				log.Fatalf(" Loader.load.readtopo error: %v", err)
			}
		} else {
			if err := t.New(l.Fhdem); err != nil {
				log.Fatalf(" Loader.load.readtopo tem.New() error: %v", err)
			}
			if err := t.Save(l.Fhdem + ".TEM.gob"); err != nil {
				log.Fatalf(" Loader.load.readtopo tem.Save() error: %v", err)
			}
		}
		tt.Lap("topo loaded")

		if _, ok := mmio.FileExists(l.Fhdem + ".ContributingCellMap.gob"); ok {
			var err error
			ucnt, err = mmio.LoadGOB(l.Fhdem + ".ContributingCellMap.gob")
			if err != nil {
				log.Fatalf(" Loader.load.readtopo error: %v", err)
			}
		} else {
			ucnt = t.ContributingCellMap()
			if err := mmio.SaveGOB(l.Fhdem+".ContributingCellMap.gob", ucnt); err != nil {
				log.Fatalf(" topo.ContributingCellMap error: %v", err)
			}
		}
		tt.Lap("topo.ContributingCellMap loaded")
	}
	readLU := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		if _, ok := mmio.FileExists(l.Flu); ok {
			fmt.Printf(" loading: %s\n", l.Flu)
			var g grid.Indx
			g.LoadGDef(gd)
			g.NewShort(l.Flu, false)
			ulu := g.UniqueValues()
			lu = *lusg.LoadLandUse(ulu)
			ilu = g.Values()
			tt.Lap("LU loaded")
		} else {
			if len(l.Flu) > 0 {
				log.Fatalf(" file not found: %s\n", l.Flu)
			}
			lu = *lusg.LoadLandUse([]int{-1})
			ilu = make(map[int]int, gd.Nactives())
			for _, c := range gd.Actives() {
				ilu[c] = -1
			}
			tt.Lap("(uniform) LU loaded")
		}
	}
	readSG := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		if _, ok := mmio.FileExists(l.Fsg); ok {
			fmt.Printf(" loading: %s\n", l.Fsg)
			var g grid.Indx
			g.LoadGDef(gd)
			g.NewShort(l.Fsg, false)
			usg := g.UniqueValues()
			sg = *lusg.LoadSurfGeo(usg)
			isg = g.Values()
			tt.Lap("SG loaded")
		} else {
			if len(l.Fsg) > 0 {
				log.Fatalf(" file not found: %s\n", l.Fsg)
			}
			sg = *lusg.LoadSurfGeo([]int{-1})
			isg = make(map[int]int, gd.Nactives())
			for _, c := range gd.Actives() {
				isg[c] = -1
			}
			tt.Lap("(uniform) SG loaded")
		}
	}
	readSWS := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.Fsws)
		sws, dsws, swscidxr = loadSWS(gd, l.Fsws)
		tt.Lap("SWS loaded")
	}

	wg.Add(3)
	go readtopo()
	go readLU()
	go readSG()
	if len(l.Fsws) > 0 {
		wg.Add(1)
		go readSWS()
	}
	wg.Wait()

	// compute static variables
	cid0, nc := -1, gd.Nactives()
	if frc != nil {
		if frc.h.Nloc() == 1 && frc.h.LocationCode() > 0 {
			cid0 = int(frc.h.Locations[0][0].(int32)) // gauge outlet id found in met file
		} else {
			log.Fatalf(" Loader.load error: unrecognized .met type\n")
		}
		if cid0 >= 0 {
			if _, ok := t.TEC[cid0]; ok {
				nc = t.UpCnt(cid0) // recount number of cells
			} else {
				cid0 = -1
			}
		}
	}

	var sif map[int][366]float64
	buildSolIrradFrac := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		fmt.Printf(" building potential solar irradiation field..\n")
		siffp := l.Fhdem
		if cid0 >= 0 {
			siffp += fmt.Sprintf(".%d", cid0)
		}
		if _, ok := mmio.FileExists(siffp + ".sif.gob"); ok {
			var err error
			sif, err = sifLoad(siffp + ".sif.gob")
			if err != nil {
				log.Fatalf(" Loader.load.buildSolIrradFrac error: %v", err)
			}
		} else {
			sif = loadSolIrradFrac(frc, &t, gd, nc, cid0, buildEp)
			if cid0 >= 0 {
				tt.Lap("PSI built, saving to gob")
				if err := sifSave(siffp+".sif.gob", sif); err != nil {
					log.Fatalf(" Loader.load.buildSolIrradFrac sif save error: %v", err)
				}
			}
		}
		tt.Lap("SolIrrad loaded")
	}

	var obs []int
	collectObs := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		if len(l.Fobs) > 0 {
			var err error
			obs, err = mmio.ReadInts(l.Fobs)
			if err != nil {
				log.Fatalf(" Loader.load.collectObs error: %v", err)
			}
			tt.Lap("collectObs complete")
		}
	}

	wg.Add(2)
	go buildSolIrradFrac()
	go collectObs()
	wg.Wait()

	mdl := STRC{
		t: &t,
		f: sif,
		u: ucnt,
		a: gd.CellArea(),
		w: gd.CellWidth(),
	}
	mpr := MAPR{
		lu:  lu,
		sg:  sg,
		ilu: ilu,
		isg: isg,
	}
	rtr := RTR{
		sws:      sws,
		dsws:     dsws,
		swscidxr: swscidxr,
	}

	return frc, mdl, mpr, rtr, gd, obs
}
