package main

import (
	"fmt"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/prep"
)

const (
	gobDir = "M:/OWRC-RDRR/owrc."
	gdefFP = "M:/OWRC-RDRR/owrc20-50a.uhdem.gdef"
	demFP  = "M:/OWRC-RDRR/owrc20-50a.uhdem"
	swsFP  = "M:/OWRC-RDRR/owrc20-50a_SWS10.indx"
	topoFP = "M:/OWRC-RDRR/owrc20-50a_SWS10.topo"
	ncfp   = "M:/OWRC-RDRR/met/202010010100.nc.bin" // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)

	lufp = "M:/OWRC-RDRR/build/lusg/solrisv3_10.bil"
	sgfp = "M:/OWRC-RDRR/build/lusg/OGSsurfGeo_50.bil"
)

var (
	dtb = time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
	dte = time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
)

func main() {

	tt := mmio.NewTimer()
	defer tt.Print("\n\nprep complete!")

	fmt.Println("\ncollecting DEM..")
	strc, gd, cells, sws, nsws := prep.BuildSTRC(gobDir, gdefFP, demFP, swsFP)

	if _, ok := mmio.FileExists(gobDir + "FORC.gob"); !ok {
		fmt.Println("\ncollecting station data and computing basin atmospheric yield and Eao..")
		prep.BuildFORC(gobDir, ncfp, cells, dtb, dte)
	}

	if _, ok := mmio.FileExists(gobDir + "RTR.gob"); !ok {
		fmt.Println("\nbuilding subbasin routing scheme..")
		prep.BuildRTR(gobDir, topoFP, strc, sws, nsws)
	}

	if _, ok := mmio.FileExists(gobDir + "MAPR.gob"); !ok {
		fmt.Println("\nbuilding land use and surficial geology mapping..")
		prep.BuildMAPR(gobDir, lufp, sgfp, gd)
		// mmio.WriteIMAP(gobDir+"luid.imap", m.LUx)
		// mmio.WriteIMAP(gobDir+"sgid.imap", m.SGx)
	}

}

// func saveGOB(fp string, d [][]float64) error {
// 	f, err := os.Create(fp)
// 	defer f.Close()
// 	if err != nil {
// 		return err
// 	}
// 	enc := gob.NewEncoder(f)
// 	err = enc.Encode(d)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func saveGOBdts(fp string, dts []time.Time) error {
// 	f, err := os.Create(fp)
// 	defer f.Close()
// 	if err != nil {
// 		return err
// 	}
// 	enc := gob.NewEncoder(f)
// 	err = enc.Encode(dts)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func saveGOBxr(fp string, xr map[int]int) error {
// 	f, err := os.Create(fp)
// 	defer f.Close()
// 	if err != nil {
// 		return err
// 	}
// 	enc := gob.NewEncoder(f)
// 	err = enc.Encode(xr)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
