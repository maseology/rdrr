package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
	"github.com/maseology/rdrr/postpro"
)

func main() {

	const (
		// mdlPrfx = "S:/OWRC-RDRR/owrc."
		mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21." //"S:/Peel/PWRMM21." //
		cid0    = 1552736
		obsfp   = "M:/Peel/RDRR-PWRMM21/dat/obs/02HB008.csv" //"S:/Peel/obs/02HB008.csv" //
	)

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	model.LoadMasterDomain(mdlPrfx)
	tt.Print("Master Domain Load complete\n")

	model.DeleteMonitors(mdlPrfx+"out/", true) // also sets-up the output folder
	if err := model.MasterDomain.Frc.AddObservation(obsfp, cid0); err != nil {
		log.Fatalln(err)
	}

	fobs := model.MasterDomain.Frc.O[0]
	oobs, _ := postpro.GetObservations("C:/Users/Mason/Desktop/", "")
	ooobs := oobs[1552736]
	cobs, ii := make([]float64, len(fobs)), 0
	dtb := time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
	dte := time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
	for i, t := range ooobs.T {
		if t.Before(dtb) || t.After(dte) {
			continue
		}
		cobs[ii] = ooobs.V[i]
		ii++
	}
	_ = fobs
	_ = cobs
	fmt.Println("")

	// // // run model
	// // TMQm := 6.7
	// // grdMin := .5
	// // kstrm := .995
	// // mcasc := 3. // .001-10
	// // soildepth := .815
	// // kfact := .088
	// // dinc := 8.5
	// // fmt.Println(model.RunDefault(mdlPrfx+"out/", mdlPrfx+"check/", TMQm, grdMin, kstrm, mcasc, soildepth, kfact, dinc, cid0, true))
	// fmt.Println(model.RunDefault(mdlPrfx+"out/", mdlPrfx+"check/", 81.625586, 0.002568, 0.497146, 0.735101, 1.213881, 0.000141, 9.854899, 1552736, true))

	// // sample models
	// model.PrepMC(mdlPrfx + "MC/")
	// model.SampleMaster(mdlPrfx, 700, cid0)

	// // find optimal model
	// model.OptimizeDefault(nil, 1104986)
}
