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
// 	gobDir = "S:/OWRC-RDRR/owrc."
// 	gdefFP = "S:/OWRC-RDRR/prep/owrc20-50a.uhdem.gdef"
// 	demFP  = "S:/OWRC-RDRR/prep/owrc20-50a.uhdem"
// 	swsFP  = "S:/OWRC-RDRR/prep/owrc20-50a_SWS10.indx"
// 	topoFP = "S:/OWRC-RDRR/prep/owrc20-50a_SWS10.topo"
// 	ncfp   = "S:/OWRC-RDRR/prep/met/202010010100.nc.bin" // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)
// 	lufprfx   = "S:/OWRC-RDRR/prep/solrisv3_10_infilled.bil"
// 	sgfp   = "S:/OWRC-RDRR/prep/OGSsurfGeo_50.bil"
// )

const (
	gobDir  = "M:/Peel/RDRR-PWRMM21/PWRMM21."
	gdefFP  = "M:/Peel/RDRR-PWRMM21/dat/elevation.real_SWS10.indx.gdef"
	demFP   = "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem"
	swsFP   = "M:/Peel/RDRR-PWRMM21/dat/elevation.real_SWS10.indx"
	topoFP  = "M:/Peel/RDRR-PWRMM21/dat/elevation.real_SWS10.topo"
	midFP   = "M:/Peel/RDRR-PWRMM21/dat/owrc20-50a_SWS10_resmpl.indx" // index meteo timeseries
	ncfp    = "M:/OWRC-RDRR/met/202010010100.nc.bin"                  // needed to convert nc to bin using /@dev/python/src/FEWS/netcdf/ncToMet.py; I cannot get github.com/fhs/go-netcdf to work on windows (as of 201027)
	lufprfx = "M:/Peel/RDRR-PWRMM21/dat/solrisv3_10_infilled.bil"
	sgfp    = "M:/Peel/RDRR-PWRMM21/dat/OGSsurfGeo_50_resmpl.indx"
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
		csws, swsc := func() (map[int]int, map[int][]int) {
			var gsws grid.Indx
			gsws.LoadGDef(gd)
			gsws.New(swsFP, false)
			cs := gsws.Values()
			sc := make(map[int][]int, len(gsws.UniqueValues()))
			for c, s := range cs {
				if _, ok := sc[s]; ok {
					sc[s] = append(sc[s], c)
				} else {
					sc[s] = []int{c}
				}
			}
			return cs, sc
		}()

		strc, cells := prep.BuildSTRC(gd, csws, gobDir, demFP)

		if _, ok := mmio.FileExists(midFP); ok { // else swsID used in place
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
			prep.BuildFORC(gobDir, ncfp, cells, dtb, dte)
		}

		if _, ok := mmio.FileExists(gobDir + "RTR.gob"); !ok {
			fmt.Println("\nbuilding subbasin routing scheme..")
			prep.BuildRTR(gobDir, topoFP, strc, swsc, csws)
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
