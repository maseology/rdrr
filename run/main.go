package main

import (
	"fmt"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/basin"
)

func main() {
	// global
	// mmio.NewTrace()
	// defer mmio.EndTrace()

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// indir := "S:/ormgp_rdrr/"
	// ldr := basin.Loader{
	// 	Dir:   indir,
	// 	Fmet:  "gob",
	// 	Fgd:   indir + "ORMGP_50_hydrocorrect.uhdem.gdef",
	// 	Fhdem: indir + "ORMGP_50_hydrocorrect.uhdem",
	// 	Flu:   indir + "ORMGP_50_hydrocorrect_SOLRISv2_ID.grd",
	// 	Fsg:   indir + "ORMGP_50_hydrocorrect_PorousMedia_ID.grd",
	// 	Fsws:  indir + "ORMGP_50_hydrocorrect_SWS10_merged.indx",
	// 	Fobs:  indir + "ORMGP_50_hydrocorrect.uhdem.obs",
	// }

	ldr := basin.Loader{
		Dir:   "M:/Peel/RDRR-PWRMM21/",
		Fmet:  "gob",
		Fgd:   "M:/Peel/RDRR-PWRMM21/dat/elevation.real_SWS10-select.indx.gdef",
		Fhdem: "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem", //"M:/Peel/Raven-PWRMM21/dat/elevation.real.hdem",
		Flu:   "M:/Peel/Raven-PWRMM21/shapefiles/solrisID.indx",
		Fsg:   "M:/Peel/Raven-PWRMM21/shapefiles/surfgeoID.indx",
		Fsws:  "M:/Peel/RDRR-PWRMM21/dat/elevation.real_SWS10-select.indx",
		Fobs:  "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem.gauges_final.obs",
	}

	// load data
	basin.LoadMasterDomain(&ldr, true)
	tt.Print("Master Domain Load complete\n")

	// run model
	basin.DeleteMonitors(ldr.Dir + "out/") // also sets-up the output folder
	basin.RunDefault(ldr.Dir, "", ldr.Dir+"check/", .045394, .004987, .116692, .073995, 1., true)

	// basin.OptimizeDefault("")

	// // sample models
	// basin.PrepMC(ldr.Dir + "MC/")
	// basin.SampleMaster(ldr.Dir, 500)
}
