package main

import (
	"fmt"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/basin"
)

const (
	mdlPrfx = "S:/OWRC-RDRR/owrc."
)

func main() {

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	basin.LoadMasterDomain(mdlPrfx)
	tt.Print("Master Domain Load complete\n")

	// run model
	basin.DeleteMonitors(mdlPrfx + "out/") // also sets-up the output folder
	topm, smax, dinc, soildepth, kfact := .045394, .004987, .116692, .073995, 1.
	basin.RunDefault(mdlPrfx, mdlPrfx+"check/", topm, smax, dinc, soildepth, kfact, true)

	// basin.OptimizeDefault("")

	// // sample models
	// basin.PrepMC(mdlPrfx + "MC/")
	// basin.SampleMaster(mdlPrfx, 500)
}
