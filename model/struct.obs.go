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
	Td                  []time.Time // [date ID]
	Oq                  [][]float64 // observed discharge (use Oxr for cross-reference)
	Oqxr, txr, mons, mt []int       // mapping of outlet cell ID to Oq[][]; other cell IDs to montior; month [1,12] cross-reference
	cellarea            float64     // (uniform) cell area and timestep
}

func dayDate(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local).Unix()
}

func collectOBS(frc *FORC, mdlprfx string, cellarea float64) *OBS {
	var mons []int
	if _, ok := mmio.FileExists(mdlprfx + "obs"); ok {
		var err error
		if mons, err = mmio.ReadInts(mdlprfx + "obs"); err != nil {
			log.Fatalf("%v", err)
		}
	}

	mt, dicT := make([]int, len(frc.T)), make(map[int64][]int)
	for k, t := range frc.T {
		mt[k] = int(t.Month())
		dicT[dayDate(t)] = append(dicT[dayDate(t)], k)
	}
	lstT, td, tx := make([]int64, 0, len(dicT)), make([]time.Time, len(dicT)), make([]int, len(frc.T))
	for u := range dicT {
		lstT = append(lstT, u)
	}
	sort.Slice(lstT, func(i, j int) bool { return lstT[i] < lstT[j] })
	for i, ud := range lstT {
		td[i] = time.Unix(ud, 0)
		if a, ok := dicT[ud]; ok {
			for _, ii := range a {
				tx[ii] = i
			}
		} else {
			panic("collectOBS error")
		}
	}

	return &OBS{
		Td:       td,
		Oq:       make([][]float64, 0),
		Oqxr:     make([]int, 0),
		txr:      tx,
		mt:       mt,
		mons:     mons,
		cellarea: cellarea,
		// dayfraction: frc.IntervalSec / 84600.,
	}
}

// AddFluxCsv reads csv file of "Date","Flow","Flag"
func (obs *OBS) AddFluxCsv(csvdir string, cxr map[int]int) {
	fps, err := mmio.FileListExt(csvdir, ".csv")
	if err != nil {
		panic(err)
	}
	nt := len(obs.Td)
	for _, fp := range fps {
		fn := mmio.FileName(fp, false)
		ii := strings.Index(fn, "-")
		if ii <= 0 {
			continue
			// log.Fatalf("OBS.AddFluxCsv error: can't find cid in filename: %s", fp)
		}
		cid, err := strconv.Atoi(fn[:ii])
		if err != nil {
			continue
			// log.Fatalf("OBS.AddFluxCsv error: %v", err)
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
		oi := len(obs.Oq) - 1

		cc := 0
		for i, t := range obs.Td {
			if v, ok := c[dayDate(t)]; ok {
				// obs.Oq[oi][i] = v * 86400. / cellarea // [m³/s] to [m/day]-leaving cell
				obs.Oq[oi][i] = v // [m³/s]
				cc++
			} else {
				obs.Oq[oi][i] = math.NaN()
			}
		}
		fmt.Printf(" > observation at cellID %d: %d of %d\n", cid, cc, nt)
	}
}

// ToDaily imports hyd [m/timestep]
func (obs *OBS) ToDaily(dat []float64) []float64 {
	nt := len(obs.Td)
	o := make([]float64, nt)
	for i, v := range dat {
		o[obs.txr[i]] += v
	}
	for j := range o {
		o[j] *= obs.cellarea / 86400. // [m³/s]
	}

	return o
}
