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
type Loader struct{ Dir, Fgd, Fdem, Fsws, Flu, Fsg, Fobs string }

func (l *Loader) load() (*FORC, STRC, MAPR, RTR, *grid.Definition, []int) {
	var wg sync.WaitGroup

	// import forcings
	var frc *FORC
	readmet := func() {
		defer wg.Done()
		frc, _ = loadGOBforcing(l.Dir + "met/")
	}

	// import structural data and mapping arrays
	gd, err := grid.ReadGDEF(l.Fgd, true)
	if err != nil {
		log.Fatalf(" grid.ReadGDEF: %v", err)
	}
	var t tem.TEM
	var lu lusg.LandUseColl
	var sg lusg.SurfGeoColl
	var ilu, isg, ilk, sws, dsws, ucnt map[int]int
	var swscidxr map[int][]int  // set of cells for every sws siswd{[]cellid}
	var uca map[int]map[int]int // unit contributing areas

	wg.Add(1)
	go readmet()

	readtopo := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.Fdem)
		var err error
		t, err = tem.LoadGob(l.Fdem + ".TEM.gob")
		if err != nil {
			log.Fatalf(" Loader.load.readtopo error: %v", err)
		}
		tt.Lap("topo loaded")

		ucnt, err = mmio.LoadGOB(l.Fdem + ".ContributingCellMap.gob")
		if err != nil {
			log.Fatalf(" Loader.load.readtopo error: %v", err)
		}
		tt.Lap("topo.ContributingCellMap loaded")
	}

	readLU := func() {
		const LakeID = 170 // SOLRIS
		tt := mmio.NewTimer()
		defer wg.Done()
		getLakes := func(ilu map[int]int) map[int]int {
			c := 0
			for _, v := range ilu {
				if v == LakeID {
					c++
				}
			}
			out := make(map[int]int, c)
			for k, v := range ilu {
				if v == LakeID {
					out[k] = -1
				}
			}
			return out
		}
		if _, ok := mmio.FileExists(l.Flu); ok {
			fmt.Printf(" loading: %s\n", l.Flu)
			var g grid.Indx
			func() {
				if _, ok := mmio.FileExists(l.Flu + ".gdef"); ok {
					gd1, err := grid.ReadGDEF(l.Flu+".gdef", false)
					if err != nil {
						log.Fatalf(" grid.ReadGDEF: %v", err)
					}
					g.LoadGDef(gd1)
				} else {
					g.LoadGDef(gd)
				}
			}()
			g.NewShort(l.Flu, false)
			ulu := g.UniqueValues()
			lu = *lusg.LoadLandUse(ulu)
			ilu = g.Values()
			ilk = getLakes(ilu) // collect open water cells
			tt.Lap("LU loaded")
		} else {
			if len(l.Flu) > 0 {
				log.Fatalf(" file not found: %s\n", l.Flu)
			}
			log.Fatalf(" file not found: %s\n", l.Flu)
			// lu = *lusg.LoadLandUse([]int{-1})
			// ilu = make(map[int]int, gd.Na)
			// for _, c := range gd.Sactives {
			// 	ilu[c] = -1
			// }
			// ilk = getLakes(ilu) // collect open water cells
			// tt.Lap("(uniform) LU loaded")
		}
	}

	readSG := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		if _, ok := mmio.FileExists(l.Fsg); ok {
			fmt.Printf(" loading: %s\n", l.Fsg)
			var g grid.Indx
			func() {
				if _, ok := mmio.FileExists(l.Fsg + ".gdef"); ok {
					gd1, err := grid.ReadGDEF(l.Fsg+".gdef", false)
					if err != nil {
						log.Fatalf(" grid.ReadGDEF: %v", err)
					}
					g.LoadGDef(gd1)
				} else {
					g.LoadGDef(gd)
				}
			}()
			g.NewShort(l.Fsg, false)
			usg := g.UniqueValues()
			sg = *lusg.LoadSurfGeo(usg)
			isg = g.Values()
			tt.Lap("SG loaded")
		} else {
			if len(l.Fsg) > 0 {
				log.Fatalf(" file not found: %s\n", l.Fsg)
			}
			log.Fatalf(" file not found: %s\n", l.Fsg)
			// sg = *lusg.LoadSurfGeo([]int{-1})
			// isg = make(map[int]int, gd.Na)
			// for _, c := range gd.Sactives {
			// 	isg[c] = -1
			// }
			// tt.Lap("(uniform) SG loaded")
		}
	}

	readSWS := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		if len(l.Fsws) > 0 {
			fmt.Printf(" loading: %s\n", l.Fsws)
			sws, dsws, swscidxr = loadSWS(gd, l.Fsws)
			tt.Lap("SWS loaded")
		}
	}

	wg.Add(4)
	go readtopo()
	go readLU()
	go readSG()
	go readSWS()
	wg.Wait()

	readUCA := func() {
		tt := mmio.NewTimer()
		defer wg.Done()
		fp := mmio.RemoveExtension(l.Fsws) + ".uca.gob"
		// fmt.Printf(" loading: %s\n", fp)
		fmt.Printf(" loading: %s\n", fp)
		var err error
		if uca, err = loadUCAgob(fp); err != nil {
			log.Fatalf(" loader.go loadUCAgob error: %v", err)
		}
		tt.Lap("UCA loaded")
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
	go readUCA()
	go collectObs()
	wg.Wait()

	mdl := STRC{
		TEM:   &t,
		UpCnt: ucnt,
		Acell: gd.CellArea(),
		Wcell: gd.Cw,
	}
	mpr := MAPR{
		lu:  lu,
		sg:  sg,
		ilu: ilu,
		isg: isg,
		ilk: ilk,
	}
	rtr := RTR{
		Sws:      sws,
		Dsws:     dsws,
		SwsCidXR: swscidxr,
		UCA:      uca,
	}

	return frc, mdl, mpr, rtr, gd, obs
}
