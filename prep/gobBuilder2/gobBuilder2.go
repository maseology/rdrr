package main

import (
	"fmt"
	"log"
	"rdrr/model"
	"rdrr/prep"
	"sort"
	"strconv"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
)

const controlFP = "M:/Peel/RDRR-PWRMM21/PWRMM21.rdrr"
const skipFRC = false

// // model period set gloabally
// var (
// 	dtb = time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
// 	dte = time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
// )

func main() {
	// var wg sync.WaitGroup

	tt := mmio.NewTimer()
	defer tt.Print("\n\nprep complete!")

	// get input file paths
	var gobDir, gdefFP, hdemFP, swsFP, luFPprfx, sgFP, gwzFP, midFP, ncFP string
	var cid0 int
	getFilePaths := func(rdrrFP string) {
		var err error
		ins := mmio.NewInstruct(rdrrFP)
		gobDir = ins.Param["prfx"][0]
		gdefFP = ins.Param["gdeffp"][0]
		hdemFP = ins.Param["hdemfp"][0]
		swsFP = ins.Param["swsfp"][0]
		gwzFP = ins.Param["gwzfp"][0]
		luFPprfx = ins.Param["lufp"][0]
		sgFP = ins.Param["sgfp"][0]
		if mfp, ok := ins.Param["midfp"]; ok {
			midFP = mfp[0] // cell-meteorological id
		}
		ncFP = ins.Param["ncfp"][0]
		if cid0, err = strconv.Atoi(ins.Param["cid0"][0]); err != nil {
			panic(err)
		}
	}
	getFilePaths(controlFP)

	// if _, ok := mmio.FileExists(gobDir + "domain.STRC.gob"); ok {
	// 	fmt.Println("\ngob files already exist, please delete to proceed")
	// 	return
	// }

	// get grid definition
	fmt.Println("\ncollecting grid defintion..")
	gd := func() *grid.Definition {
		gd, err := grid.ReadGDEF(gdefFP, true)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if len(gd.Sactives) <= 0 {
			log.Fatalf("error: grid definition requires active cells")
		}
		return gd
	}()

	// get model structure (eg. spatial constraints)
	// wg.Add(1)
	var strc *model.STRC
	var upslopes map[int][]int
	var outlets []int
	// go func() {
	fmt.Print("collecting DEM topography..")
	strc, upslopes, outlets = prep.BuildSTRC(gd, gobDir, hdemFP, cid0)
	fmt.Printf("  %d outlets; %d cells\n", len(outlets), len(upslopes))

	// strc.PrintAndCheck(mmio.GetFileDir(gobDir))

	// wg.Done()
	// }()

	// wg.Wait()
	// wg.Add(3)

	// go func() {
	fmt.Println("building subbasin routing scheme..")
	rtr := prep.BuildRTR(gobDir, strc, gd, swsFP)
	// wg.Done()
	// }()
	_ = rtr

	// go func() {
	fmt.Println("\nbuilding land use, surficial geology and gw zone mapping..")
	mapr := prep.BuildMAPR(gobDir, luFPprfx, sgFP, gwzFP, gd, strc, upslopes)
	// 	wg.Done()
	// }()
	_ = mapr

	// go func() {
	fmt.Println("\ncollecting basin atmospheric yield and demand..")

	// cell to met id cross reference
	cmxrBuild := func(fp string) map[int]int {
		o := make(map[int]int, gd.Nact)
		var g grid.Indx
		g.LoadGDef(gd)
		g.New(fp, false)
		m := g.Values()
		for _, cid := range strc.CIDs {
			if mm, ok := m[cid]; ok {
				o[cid] = mm
			} else {
				log.Fatalf("error reading " + fp)
			}
		}
		return o
	}
	var cmxr map[int]int // cell id to met id
	if _, ok := mmio.FileExists(midFP); ok {
		cmxr = cmxrBuild(midFP)
	} else {
		cmxr = cmxrBuild(swsFP) // else swsID used in place by default
	}

	if !skipFRC {
		forc := prep.BuildFORC(gobDir, ncFP, cmxr, outlets, strc.Wcell*strc.Wcell)

		_ = forc
	}

	func() {
		// if _, ok := mmio.FileExists(gobDir + "domain.gdef"); !ok {
		ocids := make([]int, len(strc.CIDs))
		copy(ocids, strc.CIDs)
		sort.Ints(ocids)
		gd.Sactives = ocids
		if err := gd.SaveAs(gobDir + "domain.gdef"); err != nil {
			panic(err)
		}
		// }
	}() // save gdef

	// 	wg.Done()
	// }()
	// wg.Wait()
}
