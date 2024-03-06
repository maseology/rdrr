package rdrr

import (
	"strconv"

	"github.com/maseology/goHydro/forcing"
	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
)

func BuildRDRR(controlFP string, intvl float64,
	iksat func(*grid.Definition, []int, []int) ([]float64, []int),
	xlu func(*grid.Definition, string, []int) SurfaceSet,
) (*Structure, *Mapper, *Subwatershed, *Parameter, *forcing.Forcing, string) {

	///////////////////////////////////////////////////////
	println("load .rdrr file")
	var mdlprfx, gdefFP, hdemFP, swsFP, luFP, sgFP, gwzFP, ncFP string
	var cid0 int
	func(rdrrFP string) { // getFilePaths
		var err error
		ins := mmio.NewInstruct(rdrrFP)
		mdlprfx = ins.Param["prfx"][0]
		if !(mmio.GetFileDir(mdlprfx) != "." && mmio.DirExists(mmio.GetFileDir(mdlprfx))) {
			mdlprfx = mmio.GetFileDir(rdrrFP) + "/" + mdlprfx
		}

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

		relativeFileCheck := func(fp string) string {
			if _, ok := mmio.FileExists(fp); !ok {
				rfp := mmio.GetFileDir(rdrrFP) + "/" + fp
				if _, ok := mmio.FileExists(rfp); ok {
					return rfp
				} else {
					panic(fp + " cannot be found")
				}
			}
			return fp
		}
		gdefFP = relativeFileCheck(gdefFP)
		hdemFP = relativeFileCheck(hdemFP)
		swsFP = relativeFileCheck(swsFP)
		luFP = relativeFileCheck(luFP)
		sgFP = relativeFileCheck(sgFP)
		gwzFP = relativeFileCheck(gwzFP)
		ncFP = relativeFileCheck(ncFP)
	}(controlFP)
	chkdir := mmio.GetFileDir(mdlprfx) + "/check/"
	mmio.MakeDir(chkdir)

	///////////////////////////////////////////////////////
	println("building model structure..")
	strc := buildSTRC(gdefFP, hdemFP, cid0)
	// strc.Checkandprint(chkdir)

	println(" > set grid mappings..")
	mp := strc.buildMapper(luFP, sgFP, gwzFP, iksat, xlu)
	// mp.Checkandprint(strc.GD, float64(strc.Nc), chkdir)

	println("\n > loading sub-watersheds (computational queuing)..")
	sws := strc.loadSWS(swsFP)
	sws.buildComputationalOrder1(strc.Cids, strc.Ds)
	// sws.checkandprint(strc.GD, strc.Cids, float64(strc.Nc), chkdir)

	////////////////////////////////////////
	////////////////////////////////////////

	// // re-project groundwater zones to sub-watersheds
	// println(" > re-mapping unique groundwater reservoirs to subwatersheds..")
	// mp.Fngwc, mp.Igw = sws.remapGWzones(&mp)

	println(" > parameterizing with defaults..")
	par := BuildParameters(&strc, &mp)
	// par.Checkandprint(strc.GD, mp.Mx, mp.Igw, chkdir)

	// summarize
	if len(chkdir) > 0 {
		println("\nBuild Summary\n==================================")
		strc.Checkandprint(chkdir)
		mp.Checkandprint(strc.GD, float64(strc.Nc), chkdir)
		sws.checkandprint(strc.GD, strc.Cids, float64(strc.Nc), chkdir)
		par.Checkandprint(strc.GD, mp.Mx, mp.Igw, chkdir)
	}

	frc := func(fp string) *forcing.Forcing {
		println(" > load forcings..")
		if _, ok := mmio.FileExists(fp); ok {
			frc, err := forcing.LoadGobForcing(fp)
			if err != nil {
				panic(err)
			}
			return frc
		}
		frc := forcing.GetForcings(sws.Isws, intvl, 0, ncFP, "") // sws id refers to the climate lists
		if err := frc.SaveGobForcing(fp); err != nil {
			panic(err)
		}
		return &frc
	}(mdlprfx + "forcing.gob")

	return &strc, &mp, &sws, &par, frc, mdlprfx
}
