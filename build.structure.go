package rdrr

import (
	"fmt"
	"log"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/tem"
)

func buildSTRC(gdefFP, hdemFP string, cid0 int) Structure {

	///////////////////////////////////////////////////////
	// STRUCTURE
	///////////////////////////////////////////////////////
	println(" > step 1: load grid definition with active cells defined")
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

	///////////////////////////////////////////////////////
	fmt.Printf(" > step 2: load topological DEM\n   loading: %s\n", hdemFP)
	dem := func() tem.TEM {
		var dem tem.TEM
		if err := dem.New(hdemFP); err != nil {
			log.Fatalf(" BuildSTRC tem.New() error: %v", err)
		}
		nwarn := 0
		// gmax, ngadj := math.Atan(.999), 0
		for _, i := range gd.Sactives {
			if _, ok := dem.TEC[i]; !ok {
				log.Fatalf(" BuildSTRC error, cell id %d not found in %s", i, hdemFP)
			}
			if dem.TEC[i].Z == -9999. {
				fmt.Printf("    WARNING no elevation assigned to cell %d\n", i)
				nwarn++
			}
			// if math.Tan(dem.TEC[i].G) > 1 {
			// 	fmt.Printf("    WARNING gradient adjusted to cell %d; too steep (angle = %.2f, set to %.2f)\n", i, dem.TEC[i].G, gmax)
			// 	ngadj++
			// 	m := dem.TEC[i]
			// 	m.G = gmax
			// 	dem.TEC[i] = m
			// }
		}
		// if ngadj > 0 {
		// 	fmt.Printf("    %d gradient adjustments made covering %.3f%% of model domain\n", ngadj, float64(ngadj)/float64(gd.Nact)*100)
		// 	nwarn+=ngadj
		// }
		if nwarn > 0 {
			println()
		}

		if gd.Nact != len(dem.TEC) {
			fmt.Println("    re-setting TEM to input GDEF..")
			demNew := tem.TEM{TEC: make(map[int]tem.TEC, gd.Nact), USlp: make(map[int][]int)}
			for _, i := range gd.Sactives {
				demNew.TEC[i] = dem.TEC[i]
			}
			for i, us := range dem.USlp {
				if _, ok := demNew.TEC[i]; ok {
					newUs := make([]int, 0, len(us))
					for _, u := range us {
						if _, ok := demNew.TEC[u]; ok {
							newUs = append(newUs, u)
						}
					}
					demNew.USlp[i] = newUs
				}
			}
			dem = demNew
		}

		return dem
	}()

	println(" > step 3: re-indexing grid ids to topo-safe arrays..")
	cids, ds := dem.DownslopeContributingAreaIDs(cid0)
	nc := len(cids)

	mx := make(map[int]int, nc) // grid cell id to array index
	for i, cid := range cids {
		mx[cid] = i
	}
	dnslp := func() []float64 {
		m := make(map[int]float64, dem.NumCells())
		for cid, tec := range dem.TEC {
			m[cid] = tec.G
		}
		dnslp := make([]float64, nc)
		for i, cid := range cids {
			if _, ok := m[cid]; !ok {
				panic("dnslp error")
			}
			dnslp[i] = m[cid]
		}
		return dnslp
	}
	dsx := func() []int { // convert from cell id to array index
		x := make([]int, nc)
		for i, cid := range cids {
			if vv, ok := ds[cid]; ok {
				if vv < 0 {
					x[i] = -1
				} else {
					x[i] = mx[vv]
				}
				continue
			} else if cid0 < 0 {
				x[i] = -1
				continue
			}
			panic("dsx error")
		}
		return x
	}

	upcnt := func() []int {
		m := dem.ContributingCellMap(cid0)
		upcnts := make([]int, nc)
		for i, cid := range cids {
			if n, ok := m[cid]; ok {
				upcnts[i] = n
			} else {
				panic("upcnt error")
			}
		}
		return upcnts
	}

	gd.ResetActives(cids)

	s := Structure{ // strc arrays are all 0-based indexed, cids is the key to grid cell id
		GD:       gd,      // grid definition
		Cids:     cids,    // topologically safe ordered grid cell ids
		Dwnslope: dnslp(), // steepest cell slope
		Ds:       dsx(),   // down slope cell index
		Upcnt:    upcnt(), // count of upslope cells
		Nc:       nc,      // number of model cells
	}

	return s
}
