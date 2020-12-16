package main

import (
	"fmt"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

func main() {

	// const mdlPrfx = "S:/OWRC-RDRR/owrc."
	const mdlPrfx = "S:/Peel/PWRMM21."
	// const mdlPrfx = "S:/RDRR-02HJ005/02HJ005."

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	model.LoadMasterDomain(mdlPrfx)
	tt.Print("Master Domain Load complete\n")

	// run model
	model.DeleteMonitors(mdlPrfx + "out/") // also sets-up the output folder

	// TMQm := 0.7750285044038263
	// grng := 0.07622992793762517
	// soildepth := 0.4443726415967753
	// kfact := 0.00010001334948482364
	// model.RunDefault(mdlPrfx+"check/", TMQm, grng, 0., soildepth, kfact, -1, true)

	// // find optimal model
	// model.OptimizeDefault(nil, -1)

	// sample models
	model.PrepMC(mdlPrfx + "MC/")
	model.SampleMaster(mdlPrfx, 2000, -1)

}
