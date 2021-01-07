package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

func main() {

	// m, grng, soildepth, kfact := model.Par4([]float64{0.313, 0.085, 0.583, 0.872})
	// fmt.Printf("\nfinal parameters:\n\tTMQm:=\t\t%v\n\tgrng:=\t\t%v\n\tsoildepth:=\t%v\n\tkfact:=\t\t%v\n\n", m, grng, soildepth, kfact)
	// os.Exit(2)

	// const mdlPrfx = "S:/OWRC-RDRR/owrc."
	const mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21." // "S:/Peel/PWRMM21."
	// const mdlPrfx = "S:/RDRR-02HJ005/02HJ005."

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	model.LoadMasterDomain(mdlPrfx)
	if err := model.MasterDomain.Frc.AddObservation("M:/Peel/RDRR-PWRMM21/dat/obs/02HB031.csv", 1104986); err != nil {
		log.Fatalln(err)
	}
	tt.Print("Master Domain Load complete\n")

	// run model
	model.DeleteMonitors(mdlPrfx + "out/") // also sets-up the output folder

	// TMQm := 1.4380620030367803
	// grng := 0.0019749463694001177
	// soildepth := 0.8744999999999999
	// kfact := 946.237161365793
	TMQm := .5
	grng := 0.95
	soildepth := 0.53
	kfact := 1.
	fmt.Println(model.RunDefault(mdlPrfx+"out/", mdlPrfx+"check/", TMQm, grng, 0., soildepth, kfact, 1104986, true))

	// // find optimal model
	// model.OptimizeDefault(nil, 1104986)

	// // sample models
	// model.PrepMC(mdlPrfx + "MC/")
	// model.SampleMaster(mdlPrfx, 2000, -1)

}
