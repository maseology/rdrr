package model

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/maseology/mmio"
)

// OBS holds forcing data
type OBS struct {
	Td             []time.Time // [date ID]
	Oq             [][]float64 // observed discharge (use Oxr for cross-reference)
	Oqxr, mons, mt []int       // mapping of outlet cell ID to Oq[][]; other cell IDs to montior; month [1,12] cross-reference
}

func dayDate(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix()
}

func collectOBS(frc *FORC, mdlprfx string) *OBS {
	var mons []int
	if _, ok := mmio.FileExists(mdlprfx + "obs"); ok {
		var err error
		if mons, err = mmio.ReadInts(mdlprfx + "obs"); err != nil {
			log.Fatalf("%v", err)
		}
	}

	mt, dicT := make([]int, len(frc.T)), make(map[int64]bool)
	for k, t := range frc.T {
		mt[k] = int(t.Month())
		dicT[dayDate(t)] = true
	}
	lstT, td := make([]int64, 0, len(dicT)), make([]time.Time, len(dicT))
	for u := range dicT {
		lstT = append(lstT, u)
	}
	sort.Slice(lstT, func(i, j int) bool { return lstT[i] < lstT[j] })
	for i, u := range lstT {
		td[i] = time.Unix(u, 0)
	}

	return &OBS{
		Td:   td,
		Oq:   make([][]float64, 0),
		Oqxr: make([]int, 0),
		mt:   mt,
		mons: mons,
	}
}

// AddFluxCsv reads csv file of "Date","Flow","Flag"
func (obs *OBS) AddFluxCsv(csvdir string, cxr map[int]int, cellarea float64) {
	fps, err := mmio.FileList(csvdir)
	if err != nil {
		panic(err)
	}
	nt := len(obs.Td)
	for _, fp := range fps {
		fn := mmio.FileName(fp, false)
		ii := strings.Index(fn, "-")
		if ii <= 0 {
			log.Fatalf("OBS.AddFluxCsv error: can't find cid in filename: %s", fp)
		}
		cid, err := strconv.Atoi(fn[:ii])
		if err != nil {
			log.Fatalf("OBS.AddFluxCsv error: %v", err)
		}
		if _, ok := cxr[cid]; !ok {
			continue
		}
		c, err := mmio.ReadCsvDateFloat(fp)
		if err != nil {
			log.Fatalf("OBS.AddFluxCsv error: %v", err)
		}
		obs.Oq = append(obs.Oq, make([]float64, nt))
		obs.Oqxr = append(obs.Oqxr, cid)

		cc := 0
		for i, t := range obs.Td {
			if v, ok := c[dayDate(t)]; ok {
				// obs.Oq[0][i] = v * 86400. / cellarea // [m³/s] to [m/day]-leaving cell
				obs.Oq[0][i] = v // [m³/s]
				cc++
			} else {
				obs.Oq[0][i] = math.NaN()
			}
		}
		fmt.Printf(" > observation at cellID %d: %d of %d\n", cid, cc, nt)
	}
}

func (obs *OBS) ToDaily(ts []time.Time, dat []float64) []float64 {
	nt := len(obs.Td)
	dv := make(map[int64]float64, nt)
	for i := range dat {
		dv[dayDate(ts[i])] += dat[i]
	}
	o := make([]float64, nt)
	for i, t := range obs.Td {
		if v, ok := dv[t.Unix()]; ok {
			o[i] = v
		} // else {
		// 	panic("ToDaily error")
		// }
	}
	return o
}
