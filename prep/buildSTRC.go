package prep

import (
	"fmt"
	"log"
	"math"
	"rdrr/model"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmio"
)

// // Cell collects cell cross-referencing
// type Cell struct {
// 	Ki, Cid, Sid, Mid int // array index, cell ID, sws ID, meteo ID
// 	// PSIf              [366]float64
// }

// BuildSTRC builds the structural (static) form of the model
func BuildSTRC(gd *grid.Definition, gobDir, demFP string, cid0 int) (*model.STRC, map[int][]int, []int) {

	dem := func() tem.TEM {
		if mmio.GetExtension(demFP) == ".gob" {
			t, err := tem.LoadGob(demFP)
			if err != nil {
				log.Fatalf(" BuildSTRC tem.LoadGob() error: %v", err)
			}
			return t
		}

		var dem tem.TEM
		if err := dem.New(demFP); err != nil {
			log.Fatalf(" BuildSTRC tem.New() error: %v", err)
		}
		gmax := math.Atan(.999)
		for _, i := range gd.Sactives {
			if _, ok := dem.TEC[i]; !ok {
				log.Fatalf(" BuildSTRC error, cell id %d not found in %s", i, demFP)
			}
			if dem.TEC[i].Z == -9999. {
				// log.Fatalf("no elevation assigned to cell %d", i)
				fmt.Printf(" WARNING no elevation assigned to cell %d\n", i)
			}
			if math.Tan(dem.TEC[i].G) > 1 {
				fmt.Printf(" WARNING gradient adjusted to cell %d; too steep\n", i)
				m := dem.TEC[i]
				m.G = gmax
				dem.TEC[i] = m
			}
		}
		if gd.Nact != len(dem.TEC) {
			log.Fatalf("BuildSTRC todo1")
			// d := make(map[int]tem.TEC, gd.Nact)
			// for _, i := range gd.Sactives {
			// 	d[i] = dem.TEC[i]
			// 	if !gd.IsActive(d[i].Ds) {
			// 		t := d[i]
			// 		t.Ds = -1
			// 		d[i] = t
			// 	}
			// }
			// dem.TEC = d
			// dem.BuildUpslopes()
		}
		return dem
	}()

	var strc *model.STRC
	ds := func() map[int]int {
		if _, ok := dem.USlp[cid0]; !ok && cid0 >= 0 { // 1-cell model
			strc = &model.STRC{
				DwnGrad: map[int]float64{cid0: dem.TEC[cid0].G},
				UpCnt:   map[int]int{cid0: 1},
				CIDs:    []int{cid0},
				DwnXR:   []int{-1},
				// Acell:   gd.Cwidth * gd.Cwidth,
				Wcell: gd.Cwidth,
				CID0:  cid0,
			}
			return map[int]int{cid0: -1}
		} else {
			cids, ds := dem.DownslopeContributingAreaIDs(cid0)
			dnslp := func() map[int]float64 {
				dnslp := make(map[int]float64, dem.NumCells())
				for cid, tec := range dem.TEC {
					dnslp[cid] = tec.G
				}
				return dnslp
			}()
			dsx := func() []int {
				m := make(map[int]int, len(cids))
				for i, cid := range cids {
					m[cid] = i
				}
				x := make([]int, len(cids))
				for i, cid := range cids {
					if vv, ok := ds[cid]; ok {
						if vv < 0 {
							x[i] = -1
						} else {
							x[i] = m[vv]
						}
						continue
					} else if cid0 < 0 {
						x[i] = -1
						continue
					}
					panic("dsx error")
				}
				return x
			}()

			strc = &model.STRC{
				DwnGrad: dnslp,
				UpCnt:   dem.ContributingCellMap(cid0),
				CIDs:    cids,
				DwnXR:   dsx,
				// Acell:   gd.Cwidth * gd.Cwidth,
				Wcell: gd.Cwidth,
				CID0:  cid0,
			}
			return ds
		}
	}()

	if err := strc.SaveGob(gobDir + "domain.STRC.gob"); err != nil {
		log.Fatalf(" BuildSTRC error: %v", err)
	}

	if cid0 < 0 {
		return strc, dem.USlp, dem.Outlets()
	}
	ups := make(map[int][]int, len(ds))
	for c := range ds {
		ups[c] = dem.USlp[c]
	}
	return strc, ups, []int{cid0}

}
