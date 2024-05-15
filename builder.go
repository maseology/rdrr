package rdrr

import (
	"fmt"
	"strconv"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/forcing"
)

func BuildRDRR(controlFP string, intvl float64,
	iksat func(*grid.Definition, []int, []int) ([]float64, []int),
	xlu func(*grid.Definition, string, []int) SurfaceSet,
) (*Structure, *Mapper, *Subwatershed, *Parameter, *forcing.Forcing, string) {

	///////////////////////////////////////////////////////
	println("loading .rdrr control file")
	var mdlprfx, gdefFP, hdemFP, swsFP, luFP, sgFP, gwzFP, ncfp string
	cid0, lakfrac := -1, -1.
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
		if mfp, ok := ins.Param["ncfp"]; ok {
			ncfp = mfp[0] // input climate data (netCDF)
		}

		if _, ok := ins.Param["cid0"]; ok { // outlet cell ID, <0 keeps while model domain
			if cid0, err = strconv.Atoi(ins.Param["cid0"][0]); err != nil {
				panic(err)
			}
		}
		if _, ok := ins.Param["lakefrac"]; ok {
			if lakfrac, err = strconv.ParseFloat(ins.Param["lakefrac"][0], 64); err != nil {
				panic(err)
			}
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
		if len(ncfp) > 0 {
			ncfp = relativeFileCheck(ncfp)
		}
	}(controlFP)
	chkdir := mmio.GetFileDir(mdlprfx) + "/check/"
	mmio.MakeDir(chkdir)
	chkdir += mmio.FileName(mdlprfx, true) // adding prefix

	////////////////////////////////////////
	// BUILD
	////////////////////////////////////////

	println("building model structure..")
	strc := buildSTRC(gdefFP, hdemFP, cid0)

	println(" > set grid mappings..")
	mp := strc.buildMapper(luFP, sgFP, gwzFP, iksat, xlu)

	println("\n > loading sub-watersheds (computational queuing)..")
	sws := strc.loadSWS(swsFP)
	sws.buildComputationalOrder1()

	////////////////////////////////////////
	// ADJUST
	////////////////////////////////////////

	// re-project groundwater zones to sub-watersheds
	println(" > re-mapping unique groundwater reservoirs to subwatersheds..")
	mp.Fngwc, mp.Igw = sws.remapGWzones(&mp)

	// set Lake HRUs
	if lakfrac > 0 {
		println(" > re-mapping lakes to subwatersheds..")
		sws.remapLakes(&mp, lakfrac)
	}

	////////////////////////////////////////
	// SET DEFAUTS
	////////////////////////////////////////

	println(" > parameterizing with defaults..")
	par := BuildParameters(&strc, &mp)

	////////////////////////////////////////
	// SUMMARIZE
	////////////////////////////////////////

	if len(chkdir) > 0 {
		println("\nBuilding summary rasters\n==================================")
		strc.Checkandprint(chkdir)
		mp.Checkandprint(strc.GD, float64(strc.Nc), chkdir)
		sws.checkandprint(strc.GD, strc.Cids, float64(strc.Nc), chkdir)
		par.Checkandprint(strc.GD, mp.Mx, mp.Igw, chkdir)
	}

	////////////////////////////////////////
	// CLIMATE FORCINGS
	////////////////////////////////////////

	frc := func(fp string) *forcing.Forcing {
		if len(fp) == 0 {
			return nil
		}
		if _, ok := mmio.FileExists(fp); ok {
			println("\n > load forcings..")
			frc, err := forcing.LoadGobForcing(fp)
			if err != nil {
				panic(err)
			}
			return frc
		}
		var frc forcing.Forcing
		switch mmio.GetExtension(ncfp) {
		case ".nc":
			fmt.Printf("\n > load forcings from %s..\n", ncfp)
			vars := []string{
				"water_potential_evaporation_amount", // PE
				"rainfall_amount",
				"surface_snow_melt_amount",
			}
			frc = forcing.GetForcings(sws.Isws, intvl, 0, ncfp, "", vars) // sws id refers to the climate lists
		case "":
			return nil
		default:
			fmt.Printf(" Load forcing ERROR: unknown file type: %s.  File %s not created.", ncfp, fp)
			return nil
		}
		frc.ToBil(strc.GD, strc.Cids, sws.Scis, chkdir)
		if err := frc.SaveGobForcing(fp); err != nil {
			panic(err)
		}
		return &frc
	}(mdlprfx + "forcing.gob")

	return &strc, &mp, &sws, &par, frc, mdlprfx
}
