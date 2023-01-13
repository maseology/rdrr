package prep

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
)

const fpSOLRISlookup = "E:/Sync/@gis/Landuse/SOLRIS_3.0/lookup_200731.csv"

func buildLandUseSOLRIS(gd *grid.Definition, fpSOLRIS string) (map[int]int, []int, map[int]int, map[int]float64, map[int]float64) {

	checkforfile := func(fp string) {
		if _, ok := mmio.FileExists(fp); !ok {
			log.Fatalf(" BuildMAPR.buildLandUse file not found: %s", fp)
		}
	}
	loadIndx := func(fp string) (map[int]int, []int) {
		checkforfile(fp)
		if _, ok := mmio.FileExists(fp); !ok {
			log.Fatalf(" BuildMAPR.buildLandUse file not found: %s", fp)
		}
		fmt.Printf(" loading: %s\n", fp)
		var g grid.Indx
		g.LoadGDef(gd)
		g.NewShort(fp, true)
		return g.Values(), g.UniqueValues()
	}
	iSOLRIS, _ := loadIndx(fpSOLRIS)
	checkforfile(fpSOLRISlookup)
	lut := loadLookupTable()

	nc := gd.Ncells()
	ilu, icov := make(map[int]int, nc), make(map[int]int, nc)
	fimp, ifct := make(map[int]float64, nc), make(map[int]float64, nc)
	for cid, solrisID := range iSOLRIS {
		if solrisID < 0 {
			ilu[cid] = typeID(surfaceTyp, "ShortVegetation")
			icov[cid] = typeID(canopyType, "Open")
			fimp[cid] = 0.
			ifct[cid] = 0.
		} else {
			if l, ok := lut[solrisID]; ok {
				ilu[cid] = typeID(surfaceTyp, l.surface) //  "-surfaceid.bil")
				icov[cid] = typeID(canopyType, l.canopy) //  "-canopyid.bil")
				fimp[cid] = l.perimp                     //  "-perimp.bil")
				ifct[cid] = l.percov                     //  "-percov.bil") // fraction cover (to be adjusted below)
			} else {
				panic("SOLRIS ID not found")
			}
		}

	}

	ulu := func() []int { // unique/distict values
		c, i := make([]int, len(ilu)), 0
		for _, v := range ilu {
			c[i] = v
			i++
		}
		return mmaths.UniqueInts(c)
	}()

	return ilu, ulu, icov, fimp, ifct
}

type lookup struct {
	canopy, surface string
	perimp, percov  float64
}

var surfaceTyp = []string{
	"Noflow",
	"Waterbody",
	"ShortVegetation",
	"TallVegetation",
	"Urban",
	"Agriculture",
	"Forest",
	"Meadow",
	"Wetland",
	"Swamp",
	"Marsh",
	"Channel",
	"Lake",
	"Barren",
	"SparseVegetation",
	"DenseVegetation",
}
var canopyType = []string{
	"Open",
	"Shrub",
	"Coniferous",
	"Deciduous",
	"MixedVegetation",
}

func typeID(ids []string, q string) int {
	q = strings.ToLower(q)
	for i, id := range ids {
		if strings.ToLower(id) == q {
			return i
		}
	}
	panic("BuildMAPR.typeID")
}

func loadLookupTable() map[int]lookup {
	lns, err := mmio.ReadTextLines(fpSOLRISlookup)
	if err != nil {
		log.Fatalf(" BuildMAPR.loadLookupTable file %s error\n%v", fpSOLRISlookup, err)
	}

	m := make(map[int]lookup)
	for i, ln := range lns {
		stp := strings.Split(ln, ",")
		solID, err := strconv.Atoi(stp[0])
		if err != nil {
			if i > 0 {
				panic("BuildMAPR.loadLookupTable 1")
			}
			continue
		}

		strflt := func(s string) float64 {
			if s == "" {
				return 0.
			}
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				panic(err)
			}
			return v
		}

		la := len(stp)
		m[solID] = lookup{canopy: stp[la-4], surface: stp[la-3], perimp: strflt(stp[la-2]), percov: strflt(stp[la-1])}

	}

	return m
}
