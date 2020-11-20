package prep

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/lusg"
	"github.com/maseology/rdrr/model"
)

const (
	defaultDepSto    = 0.001  // [m]
	defaultIntSto    = 0.0005 // [m]
	defaultSoilDepth = 0.1    // [m]
	defaultPorosity  = 0.2    // [-]
	defaultFc        = 0.3    // [-]
)

const ( // canopy types
	open = iota
	shrub
	coniferous
	deciduous
	mixedVegetation
)

// BuildMAPR returns (and saves) the parameter mapping scheme
func BuildMAPR(gobDir, lufp, sgfp string, gd *grid.Definition) *model.MAPR {
	var wg sync.WaitGroup
	var lu lusg.LandUseColl
	var sg lusg.SurfGeoColl
	var ilu, isg, ilk map[int]int
	var fimp, fcov map[int]float64

	readLU := func() {
		tt := mmio.NewTimer()
		defer wg.Done()

		checkforfile := func(fp string) {
			if _, ok := mmio.FileExists(fp); !ok {
				log.Fatalf(" BuildMAPR.readLU file not found: %s", fp)
			}
		}

		// load data
		loadReal := func(fp string) map[int]float64 {
			checkforfile(fp)
			fmt.Printf(" loading: %s\n", fp)
			var g grid.Real
			g.NewGD32(fp, gd)
			aout := make(map[int]float64, len(g.A))
			for k, v := range g.A {
				if v < 0. {
					aout[k] = 0.
				} else {
					aout[k] = v
				}
			}
			return aout
		}
		fimp = loadReal(lufp + "-perimp.bil")
		fcov = loadReal(lufp + "-percov.bil")

		// load indices
		loadIndx := func(fp string) (map[int]int, []int) {
			checkforfile(fp)
			if _, ok := mmio.FileExists(fp); !ok {
				log.Fatalf(" BuildMAPR.readLU file not found: %s", fp)
			}
			fmt.Printf(" loading: %s\n", fp)
			var g grid.Indx
			g.LoadGDef(gd)
			g.NewShort(fp, true)
			return g.Values(), g.UniqueValues()
		}
		var ulu []int
		ilu, ulu = loadIndx(lufp + "-surfaceid.bil")
		icov, _ := loadIndx(lufp + "-canopyid.bil")

		// adjust cover (convert to ifct)
		for k, v := range fcov {
			if ic, ok := icov[k]; ok {
				fcov[k] = v * relativeCover(ic, ilu[k])
			}
		}

		getLakes := func(ilu map[int]int) map[int]int {
			c := 0
			for _, v := range ilu {
				if v == lusg.Lake {
					c++
				}
			}
			out := make(map[int]int, c)
			for k, v := range ilu {
				if v == lusg.Lake {
					out[k] = -1
				}
			}
			return out
		}

		loadLandUseDefaults := func(UniqueValues []int) lusg.LandUseColl {
			// create LandUse collection
			p := make(map[int]lusg.LandUse, len(UniqueValues))
			for _, i := range UniqueValues {
				p[i] = lusg.LandUse{ID: i, DepSto: defaultDepSto, IntSto: defaultIntSto, SoilDepth: defaultSoilDepth, Porosity: defaultPorosity, Fc: defaultFc}
			}
			return p
		}

		lu = loadLandUseDefaults(ulu)
		ilk = getLakes(ilu) // collect open water cells
		tt.Lap("LU loaded")
	}

	readSG := func() {
		tt := mmio.NewTimer()
		defer wg.Done()

		checkforfile := func(fp string) {
			if _, ok := mmio.FileExists(fp); !ok {
				log.Fatalf(" BuildMAPR.readSG file not found: %s", fp)
			}
		}

		// load index
		loadIndx := func(fp string) (map[int]int, []int) {
			checkforfile(fp)
			if _, ok := mmio.FileExists(fp); !ok {
				log.Fatalf(" BuildMAPR.readSG file not found: %s", fp)
			}
			fmt.Printf(" loading: %s\n", fp)
			var g grid.Indx
			g.LoadGDef(gd)
			g.NewShort(fp, true)
			return g.Values(), g.UniqueValues()
		}
		var usg []int
		isg, usg = loadIndx(sgfp)
		sg = *lusg.LoadSurfGeo(usg)
		tt.Lap("SG loaded")
	}

	wg.Add(2)
	go readLU()
	go readSG()
	wg.Wait()

	mpr := model.MAPR{
		LU:   lu,
		SG:   sg,
		LUx:  ilu,
		SGx:  isg,
		LKx:  ilk,
		Fimp: fimp,
		Ifct: fcov,
	}

	if err := mpr.SaveGob(gobDir + "MAPR.gob"); err != nil {
		log.Fatalf(" BuildMAPR error: %v", err)
	}

	return &mpr
}

// relativeCover creates a canopy cover factor based on land use
func relativeCover(canopyID, surfaceID int) float64 {
	f := 0.
	switch canopyID {
	case coniferous, deciduous, mixedVegetation:
		f += 1.
	case shrub:
		f += .5
	}
	switch surfaceID {
	case lusg.DenseVegetation:
		f += 1.25
	case lusg.ShortVegetation, lusg.TallVegetation, lusg.Forest, lusg.Swamp:
		f += 1.
	case lusg.Agriculture, lusg.Meadow:
		f += .85
	case lusg.Wetland, lusg.Marsh, lusg.SparseVegetation:
		f += .35
	}
	return f
}
