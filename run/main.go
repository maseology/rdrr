package main

import (
	"fmt"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/basin"
)

func main() {

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	ldr := basin.Loader{
		Dir:  "S:/OWRC-RDRR/",
		Fgd:  "S:/OWRC-RDRR/owrc20-50a.uhdem.gdef",
		Fdem: "S:/OWRC-RDRR/owrc20-50a.uhdem",
		Flu:  "M:/Peel/Raven-PWRMM21/shapefiles/solrisID.indx",
		Fsg:  "M:/Peel/Raven-PWRMM21/shapefiles/surfgeoID.indx",
		Fsws: "S:/OWRC-RDRR/owrc20-50a_SWS10.indx",
		Fobs: "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem.gauges_final.obs",
	}

	// load data
	basin.LoadMasterDomain(&ldr)
	tt.Print("Master Domain Load complete\n")

	// run model
	basin.DeleteMonitors(ldr.Dir + "out/") // also sets-up the output folder
	topm, smax, dinc, soildepth, kfact := .045394, .004987, .116692, .073995, 1.
	basin.RunDefault(ldr.Dir, ldr.Dir+"check/", topm, smax, dinc, soildepth, kfact, true)

	// basin.OptimizeDefault("")

	// // sample models
	// basin.PrepMC(ldr.Dir + "MC/")
	// basin.SampleMaster(ldr.Dir, 500)
}
