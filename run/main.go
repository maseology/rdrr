package main

import (
	"fmt"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

const (
	mdlPrfx = "S:/OWRC-RDRR/owrc."
	obsfp   = "S:/OWRC-RDRR/owrc20-50-obs.obs"
)

func main() {

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	model.LoadMasterDomain(mdlPrfx, obsfp)
	tt.Print("Master Domain Load complete\n")

	// run model
	model.DeleteMonitors(mdlPrfx + "out/") // also sets-up the output folder
	// topm, smax, dinc, soildepth, kfact := .045394, .004987, .116692, .073995, 1.
	// topm, smax, dinc, soildepth, kfact := 0.001153, 2.287310, 0.104665, 1.435206, 33.153130
	// model.RunDefault(mdlPrfx, mdlPrfx+"check/", topm, smax, dinc, soildepth, kfact, 10658626, true)

	// model.OptimizeDefault("")

	// sample models
	model.PrepMC(mdlPrfx + "MC/")
	model.SampleMaster(mdlPrfx, 100)
}
