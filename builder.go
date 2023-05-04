package rdrr

import (
	"strconv"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
)

func BuildRDRR(controlFP string,
	iksat func([]int) []float64,
	xlu func(*grid.Definition, string, map[int]int) SurfaceSet,
) {

	///////////////////////////////////////////////////////
	println("load .rdrr file")
	var mdlprfx, gdefFP, hdemFP, swsFP, luFP, sgFP, gwzFP, ncFP string
	var cid0 int
	func(rdrrFP string) { // getFilePaths
		var err error
		ins := mmio.NewInstruct(rdrrFP)
		mdlprfx = ins.Param["prfx"][0]
		gdefFP = ins.Param["gdeffp"][0]
		hdemFP = ins.Param["hdemfp"][0]
		swsFP = ins.Param["swsfp"][0]
		sgFP = ins.Param["sgfp"][0]

		if gfp, ok := ins.Param["gwzfp"]; ok {
			gwzFP = gfp[0] // groundwater id
		}
		if lfp, ok := ins.Param["lufp"]; ok {
			luFP = lfp[0] // land-use id
		}
		ncFP = ins.Param["ncfp"][0]
		if cid0, err = strconv.Atoi(ins.Param["cid0"][0]); err != nil {
			panic(err)
		}
	}(controlFP)

	///////////////////////////////////////////////////////
	println("building..")
	chkdir := mmio.GetFileDir(mdlprfx) + "/check/"
	strc := buildSTRC(gdefFP, hdemFP, cid0)

	println("  loading sub-watersheds (computational queuing)..")
	sws := strc.loadSWS(swsFP)
	sws.buildComputationalOrder1(strc.Cids, strc.Ds)

	println("  set grid mappings..")
	mp := strc.buildMapper(luFP, sgFP, gwzFP, iksat, xlu)

	// re-project groundwater zones to sub-watersheds
	mp.Fngwc, mp.Igw = sws.remapGWzones(&mp)

	frc := func(fp string) *Forcing {
		if _, ok := mmio.FileExists(fp); ok {
			frc, err := LoadGobForcing(fp)
			if err != nil {
				panic(err)
			}
			return frc
		}
		println("  load forcings..")
		frc := buildForcings(sws.Isws, ncFP) // sws id refers to the climate lists
		if err := frc.saveGob(fp); err != nil {
			panic(err)
		}
		return &frc
	}(mdlprfx + "forcing.gob")
	_ = frc

	println("  parameterizing with defaults..")
	par := buildParameters(&strc, &mp)

	func() {
		fltr := grid.FilterGaussianSmoothing // 5x5 matrix
		zetaConv := make([]float64, len(par.Zeta))
		for _, cid := range strc.GD.Sactives {
			if k, ok := mp.Mx[cid]; ok {
				r, c := strc.GD.RowCol(cid)
				dnm := 0.
				for m := -2; m <= 2; m++ {
					for n := -2; n <= 2; n++ {
						if bcid := strc.GD.CellID(r+m, c+n); bcid >= 0 {
							if kk, ok := mp.Mx[bcid]; ok {
								zetaConv[k] += par.Zeta[kk] * fltr[m+2][n+2]
								dnm += fltr[m+2][n+2]
							}
						}
					}
				}
				zetaConv[k] /= dnm
			}
		}
		for i, v := range zetaConv {
			par.Zeta[i] = v
		}
	}()

	// summarize
	if len(chkdir) > 0 {
		println("\nBuild Summary\n==================================")
		strc.checkandprint(chkdir)
		mp.checkandprint(strc.GD, float64(strc.Nc), chkdir)
		sws.checkandprint(strc.GD, strc.Cids, float64(strc.Nc), chkdir)
		par.checkandprint(strc.GD, mp.Mx, mp.Igw, chkdir)
	}

	// save gobs
	println("\nSaving gobs..")
	if err := strc.GD.SaveAs(mdlprfx + "gdef"); err != nil {
		panic(err)
	}
	if err := strc.saveGob(mdlprfx + "structure.gob"); err != nil {
		panic(err)
	}
	if err := mp.saveGob(mdlprfx + "mapper.gob"); err != nil {
		panic(err)
	}
	if err := sws.saveGob(mdlprfx + "subwatershed.gob"); err != nil {
		panic(err)
	}
	if err := par.saveGob(mdlprfx + "parameter.gob"); err != nil {
		panic(err)
	}

	println()
}
