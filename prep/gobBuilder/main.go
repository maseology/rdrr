package main

import (
	"fmt"
	"log"
	"time"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/prep"
)

// const (
// 	gobDir  = "M:/OWRC-RDRR/owrc."
// 	gdefFP  = "M:/OWRC-RDRR/owrc20-50a.uhdem.gdef"
// 	demFP   = "M:/OWRC-RDRR/owrc20-50a.uhdem"
// 	swsFP   = "M:/OWRC-RDRR/owrc20-50a_SWS10.indx"
// 	lufprfx = "M:/OWRC-RDRR/solrisv3_10_infilled.bil"
// 	sgfp    = "M:/OWRC-RDRR/build/lusg/OGSsurfGeo_50.bil"
// 	ncfp    = "M:/OWRC-RDRR/met/202010010100.nc.bin" // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)
// 	midFP   = ""                                     // dummy, swsFP wil be used instead
// )
// const (
// 	gobDir  = "M:/RDRR-02HJ005/02HJ005."
// 	gdefFP  = "M:/RDRR-02HJ005/dat/02HJ005.gdef"
// 	demFP   = "M:/RDRR-02HJ005/dat/owrc20-50a-elevation_resmpl.uhdem"
// 	swsFP   = "M:/RDRR-02HJ005/dat/owrc20-50a-elevation_resmpl.real_SWS10.indx"
// 	lufprfx = "M:/RDRR-02HJ005/dat/solrisv3_10_infilled.bil"
// 	sgfp    = "M:/RDRR-02HJ005/dat/OGSsurfGeo_50_resmpl.indx"
// 	midFP   = "M:/RDRR-02HJ005/dat/owrc20-50a_SWS10_resmpl.indx" // index meteo timeseries
// 	ncfp    = "M:/OWRC-RDRR/met/202010010100.nc.bin"             // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)
// )
// const (
// 	gobDir  = "M:/RDRR-02HK016/02HK016."
// 	gdefFP  = "M:/RDRR-02HK016/dat/02HK016_50.uhdem.gdef"
// 	demFP   = "M:/RDRR-02HK016/dat/02HK016_50.uhdem"
// 	swsFP   = "M:/RDRR-02HK016/dat/02HK016_50_SWS10.indx"
// 	lufprfx = "M:/RDRR-02HK016/dat/solrisv3_10_infilled.bil"
// 	sgfp    = "M:/RDRR-02HK016/dat/OGSsurfGeo_50_resmpl.indx"
// 	midFP   = "M:/RDRR-02HK016/dat/owrc20-50a_SWS10_resmpl.indx" // index meteo timeseries
// 	ncfp    = "M:/OWRC-RDRR/met/202010010100.nc.bin"             // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)
// )
const (
	gobDir  = "M:/Peel/RDRR-PWRMM21/PWRMM21."
	gdefFP  = "M:/Peel/RDRR-PWRMM21/dat/elevation.real_SWS10.indx.gdef"
	demFP   = "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem"
	swsFP   = "M:/Peel/RDRR-PWRMM21/dat/elevation.real_SWS10.indx"
	midFP   = "M:/Peel/RDRR-PWRMM21/dat/owrc20-50a_SWS10_resmpl.indx" // index meteo timeseries
	lufprfx = "M:/Peel/RDRR-PWRMM21/dat/solrisv3_10_infilled.bil"
	sgfp    = "M:/Peel/RDRR-PWRMM21/dat/OGSsurfGeo_50_resmpl.indx"
	ncfp    = "M:/OWRC-RDRR/met/202010010100.nc.bin" // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)
)

var (
	dtb = time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
	dte = time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
)

func main() {

	tt := mmio.NewTimer()
	defer tt.Print("\n\nprep complete!")

	// get grid definition
	fmt.Println("\ncollecting grid defintion..")
	gd := func() *grid.Definition {
		gd, err := grid.ReadGDEF(gdefFP, true)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if len(gd.Sactives) <= 0 {
			log.Fatalf("error: grid definition requires active cells")
		}
		return gd
	}()

	if _, ok := mmio.FileExists(gobDir + "STRC.gob"); !ok {
		fmt.Println("collecting DEM and subwatersheds..")

		csws, dsws, swsc := prep.CollectSWS(swsFP, gd)

		strc, cells := prep.BuildSTRC(gd, csws, gobDir, demFP)

		if _, ok := mmio.FileExists(midFP); ok { // else swsID used in place by default
			var gmid grid.Indx
			gmid.LoadGDef(gd)
			gmid.New(midFP, false)
			m := gmid.Values()
			for k, c := range cells {
				if mm, ok := m[c.Cid]; ok {
					cells[k].Mid = mm
				} else {
					log.Fatalf("error reading " + midFP)
				}
			}
		}

		if _, ok := mmio.FileExists(gobDir + "FORC.gob"); !ok {
			fmt.Println("\ncollecting station data and computing basin atmospheric yield and Eao..")
			outlets := strc.TEM.Outlets()
			prep.BuildFORC(gobDir, ncfp, cells, dtb, dte, outlets, strc.Acell)
		}

		if _, ok := mmio.FileExists(gobDir + "RTR.gob"); !ok {
			fmt.Println("\nbuilding subbasin routing scheme..")
			prep.BuildRTR(gobDir, strc, csws, dsws, len(swsc))
		}
	}

	if _, ok := mmio.FileExists(gobDir + "MAPR.gob"); !ok {
		fmt.Println("\nbuilding land use and surficial geology mapping..")
		prep.BuildMAPR(gobDir, lufprfx, sgfp, gd)
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
