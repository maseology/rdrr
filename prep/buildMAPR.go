package prep

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/lusg"
	"github.com/maseology/rdrr/model"
)

const ( // canopy types
	open = iota
	shrub
	coniferous
	deciduous
	mixedVegetation
)

// BuildMAPR returns (and saves) the parameter mapping scheme
func BuildMAPR(gobDir, lufp, sgfp, gwfp string, gd *grid.Definition, strc *model.STRC, upslopes map[int][]int) *model.MAPR {
	var wg sync.WaitGroup
	// var lu lusg.LandUseColl
	// var gw map[int]lusg.TOPMODEL
	var ilu, isg, igw map[int]int
	var ksat, fimp, ifct map[int]float64
	var gwuca, fngwc []float64

	// collect stream cells
	strms, _ := buildStreams(strc, strc.CIDs)

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
		fimp = loadReal(lufp + "-perimp.bil")
		ifct = loadReal(lufp + "-percov.bil") // fraction cover (to be adjusted below)

		// adjust cover (convert to ifct)
		for k, v := range ifct {
			if ic, ok := icov[k]; ok {
				ifct[k] = v * relativeCover(ic, ilu[k])
			}
		}

		// force stream cells to Channel type
		for _, c := range strms {
			ilu[c] = lusg.Channel
		}
		if func() bool { // check if channels allready exist in ulu, if not, add
			for _, c := range ulu {
				if c == lusg.Channel {
					return false
				}
			}
			return true
		}() {
			ulu = append(ulu, lusg.Channel)
		}

		// getLUtypes := func(ilu map[int]int, LUtype int) map[int]int {
		// 	c := 0
		// 	for _, v := range ilu {
		// 		if v == LUtype {
		// 			c++
		// 		}
		// 	}
		// 	out := make(map[int]int, c)
		// 	for k, v := range ilu {
		// 		if v == LUtype {
		// 			out[k] = -1
		// 		}
		// 	}
		// 	return out
		// }

		// loadLandUseDefaults := func(UniqueValues []int) lusg.LandUseColl {
		// 	// create LandUse collection
		// 	p := make(map[int]lusg.LandUse, len(UniqueValues))
		// 	for _, i := range UniqueValues {
		// 		p[i] = lusg.LandUse{
		// 			DepSto:   defaultDepSto,
		// 			IntSto:   defaultIntSto,
		// 			Porosity: defaultPorosity,
		// 			Fc:       defaultFc,
		// 		}
		// 	}
		// 	return p
		// }

		// lu = loadLandUseDefaults(ulu)
		// ilk = getLakes(ilu, lusg.Waterbody) // collect open water cells
		tt.Lap("LU loaded")
	}

	readSG := func() {
		tt := mmio.NewTimer()
		defer wg.Done()

		// load index
		loadIndx := func(fp string) (map[int]int, []int) {
			if _, ok := mmio.FileExists(fp); !ok {
				log.Fatalf(" BuildMAPR.readSG file not found: %s", fp)
			}
			fmt.Printf(" loading: %s\n", fp)
			var g grid.Indx
			switch mmio.GetExtension(fp) {
			case ".bil":
				g.LoadGDef(gd)
				g.NewShort(fp, true)
			case ".indx":
				if _, b := mmio.FileExists(fp + ".gdef"); !b {
					g.LoadGDef(gd)
				}
				g.New(fp, true)
			default:
				log.Fatalf("unrecognized file format: " + fp)
			}
			return g.Values(), g.UniqueValues()
		}
		var usg []int
		isg, usg = loadIndx(sgfp)
		ksat = lusg.LoadKsat(usg)
		tt.Lap("SG loaded")
	}

	readGW := func() {
		tt := mmio.NewTimer()
		defer wg.Done()

		// load index
		loadIndx := func(fp string) (map[int]int, []int) {
			if _, ok := mmio.FileExists(fp); !ok {
				log.Fatalf(" BuildMAPR.readGW file not found: %s", fp)
			}
			fmt.Printf(" loading: %s\n", fp)
			var g grid.Indx
			switch mmio.GetExtension(fp) {
			case ".bil":
				g.LoadGDef(gd)
				g.NewShort(fp, true)
			case ".indx":
				if _, b := mmio.FileExists(fp + ".gdef"); !b {
					g.LoadGDef(gd)
				}
				g.New(fp, true)
			default:
				log.Fatalf("unrecognized file format: " + fp)
			}
			return g.Values(), g.UniqueValues()
		}
		mgw, _ := loadIndx(gwfp)
		gwuca, fngwc, igw = buildGWzone(strc, upslopes, mgw, strc.CIDs)
		tt.Lap("GW zones loaded")
	}

	wg.Add(3)
	go readLU()
	go readSG()
	go readGW()
	wg.Wait()

	// mpr := model.MAPR{
	// 	LU:   lu,
	// 	GW:   gw,
	// 	LUx:  ilu,
	// 	SGx:  isg,
	// 	GWx:  igw,
	// 	Ksat: ksat,
	// 	Fimp: fimp,
	// 	Ifct: ifct,
	//  Strms: strms,
	// }

	// subsetting
	mprSS := func() *model.MAPR {
		ncid := len(strc.CIDs)
		iluSS, isgSS, igwSS := make(map[int]int, ncid), make(map[int]int, ncid), make(map[int]int, ncid)
		ksatSS, ucaSS := map[int]float64{}, make(map[int]float64, ncid)
		fimpSS, ifctSS := make(map[int]float64, ncid), make(map[int]float64, ncid)

		// luSS, gwSS := lusg.LandUseColl{}, lusg.GWzoneColl{}
		// gwSS := map[int]lusg.TOPMODEL{}

		for i, c := range strc.CIDs {
			iluSS[c] = ilu[c]
			isgSS[c] = isg[c]
			igwSS[c] = igw[c]
			fimpSS[c] = fimp[c]
			ifctSS[c] = ifct[c]
			ucaSS[c] = gwuca[i]
			// if _, ok := luSS[iluSS[c]]; !ok {
			// 	luSS[iluSS[c]] = lu[iluSS[c]]
			// }
			// if _, ok := gwSS[igwSS[c]]; !ok {
			// 	gwSS[igwSS[c]] = gw[igwSS[c]]
			// }
			if _, ok := ksatSS[isgSS[c]]; !ok {
				ksatSS[isgSS[c]] = ksat[isgSS[c]]
			}
		}

		return &model.MAPR{
			// LU:   luSS,
			// GW:    gwSS,
			LUx:   iluSS,
			SGx:   isgSS,
			GWx:   igwSS,
			Ksat:  ksatSS,
			Fimp:  fimpSS,
			Ifct:  ifctSS,
			Strms: strms,
			Uca:   ucaSS,
			Fngwc: fngwc,
		}
	}()

	if err := mprSS.SaveGob(gobDir + "domain.MAPR.gob"); err != nil {
		log.Fatalf(" BuildMAPR error: %v", err)
	}

	return mprSS
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

func buildGWzone(strc *model.STRC, upslopes map[int][]int, cgw map[int]int, cids []int) (uca, fngwc []float64, agw map[int]int) {
	xgw := func() map[int]int {
		d := make(map[int]int)
		for _, c := range cids {
			d[cgw[c]]++
		}
		u := make([]int, 0, len(d))
		for k := range d {
			u = append(u, k)
		}
		sort.Ints(u)
		for i, uu := range u {
			if _, ok := d[uu]; !ok {
				panic("buildGWzone mgw error 1")
			}
			d[uu] = i
		}
		return d
	}() // gw zoned ID to 0-base array index

	mcids := make(map[int]int, len(cids))
	mx := make(map[int]int, len(cids))
	cidxr := make([][]int, len(xgw)) // [agid][cellIDs]
	for i, c := range cids {
		if _, ok := cgw[c]; !ok {
			panic("buildGWzone error")
		}
		mx[c] = i // cell id to array id
		// g := cgw[c]                    // cell id to gw zone id; here, indices must be zero-based
		g := xgw[cgw[c]]               // cell id to gw zone array id
		mcids[c] = g                   // sub-setting to cids array
		cidxr[g] = append(cidxr[g], c) // assumes 0-based indexing, may need to cross-reference //////////////////////////////////// see above
	}

	func() {
		remove := func(slice [][]int, s int) [][]int {
			return append(slice[:s], slice[s+1:]...)
		}
		s := []int{}
		for i, a := range cidxr {
			if len(a) == 0 {
				s = append(s, i)
			}
		}
		for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
			s[i], s[j] = s[j], s[i]
		}
		for _, i := range s {
			cidxr = remove(cidxr, i)
		}
	}()

	// compute unit contributing areas
	uca, fngwc = func() ([]float64, []float64) {
		fmt.Print(" building unit contributing areas.. ")
		type col struct {
			g int
			u map[int]int
		}
		ch := make(chan col, len(cidxr))

		for g, cids := range cidxr {
			go func(g int, cids []int) {
				m := make(map[int]int, len(cids))
				for _, c := range cids {
					m[c] = 1
					for _, u := range upslopes[c] {
						if cgw[u] == g { // to be kept within gw-zone
							// m[c] += strc.TEM.UnitContributingArea(u)
							m[c] += strc.UpCnt[u]
						}
					}
				}
				ch <- col{g, m}
			}(g, cids)
		}
		fngwc = make([]float64, len(cidxr))
		us := make([]map[int]int, len(cidxr))
		for i := 0; i < len(cidxr); i++ {
			c := <-ch
			us[c.g] = c.u                  // assumes 0-based indexing, may need to cross-reference //////////////////////////////////// see above
			fngwc[c.g] = float64(len(c.u)) // assumes 0-based indexing, may need to cross-reference //////////////////////////////////// see above
		}
		close(ch)

		uca, cc := make([]float64, len(cids)), 0
		for _, u := range us {
			for cid, a := range u {
				uca[mx[cid]] = float64(a)
				cc++
			}
		}
		if cc != len(cids) {
			panic("UCA error")
		}
		return uca, fngwc
	}()

	return uca, fngwc, mcids
}
