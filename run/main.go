package main

import (
	"fmt"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

func main() {

	// const (
	// 	mdlPrfx = "S:/OWRC-RDRR/owrc."
	// 	monFP   = "S:/OWRC-RDRR/owrc20-50-obs.obs"
	// )

	const mdlPrfx = "M:/RDRR-02HJ005/02HJ005."

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	model.LoadMasterDomain(mdlPrfx)
	tt.Print("Master Domain Load complete\n")

	// run model
	model.DeleteMonitors(mdlPrfx + "out/") // also sets-up the output folder

	TMQm := 0.020565
	grng := 1.656823
	soildepth := 0.370598
	kfact := 1.
	model.RunDefault(mdlPrfx+"check/", TMQm, grng, 0., soildepth, kfact, -1, true)

	// sample model
	// model.OptimizeDefault(nil, obsFP, outlet)

	// // sample models
	// model.PrepMC(mdlPrfx + "MC/")
	// model.SampleMaster(mdlPrfx, 200, outlet)

}
