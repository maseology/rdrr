package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/prep"
)

const (
	gdefFP = "S:/OWRC-RDRR/owrc20-50a.uhdem.gdef"
	demFP  = "S:/OWRC-RDRR/owrc20-50a.uhdem"
	swsFP  = "S:/OWRC-RDRR/owrc20-50a_SWS10.indx"
	ncfp   = "M:/OWRC-RDRR/met/202010010100.nc.bin" // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)

	gobDir = "M:/OWRC-RDRR/met/"
)

var (
	dtb = time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
	dte = time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
)

func main() {
	tt := mmio.NewTimer()
	defer tt.Print("prep complete!")

	fmt.Println("\ncollecting DEM..")
	prep.GetCells(gdefFP, demFP, swsFP)

	fmt.Println("\ncollecting station data and computing basin atmospheric yield and Eao..")
	dts, ys, eao, mxr, _ := prep.CollectMeteoData(ncfp, dtb, dte)
	fmt.Printf("\n Model start:\t%v\n Model end:\t%v\n saving met gobs..\n", dts[0], dts[len(dts)-1])
	if err := saveGOB(gobDir+"frc.ys.gob", ys); err != nil {
		log.Fatalf("%v", err)
	}
	if err := saveGOB(gobDir+"frc.ep.gob", eao); err != nil {
		log.Fatalf("%v", err)
	}
	if err := saveGOBxr(gobDir+"frc.xr.gob", mxr); err != nil {
		log.Fatalf("%v", err)
	}
}

func saveGOB(fp string, d [][]float64) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(d)
	if err != nil {
		return err
	}
	return nil
}

func saveGOBxr(fp string, xr map[int]int) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(xr)
	if err != nil {
		return err
	}
	return nil
}
