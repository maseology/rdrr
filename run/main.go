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

	indir := "S:/ormgp_rdrr/"
	ldr := basin.Loader{
		Dir:   indir,
		Fmet:  "gob",
		Fgd:   indir + "ORMGP_50_hydrocorrect.uhdem.gdef",
		Fhdem: indir + "ORMGP_50_hydrocorrect.uhdem",
		Flu:   indir + "ORMGP_50_hydrocorrect_SOLRISv2_ID.grd",
		Fsg:   indir + "ORMGP_50_hydrocorrect_PorousMedia_ID.grd",
		Fsws:  indir + "ORMGP_50_hydrocorrect_SWS10_merged.indx",
		Fobs:  indir + "ORMGP_50_hydrocorrect.uhdem.obs",
	}

	// load data
	basin.LoadMasterDomain(&ldr, true)
	tt.Print("Master Domain Load complete\n")
	basin.PrepMC(indir + "MC/")

	// run model
	basin.SampleMaster(indir, 500)
}
