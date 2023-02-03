package rdrr

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/maseology/mmio"
)

// AddFluxCsv reads csv file of "Date","Flow","Flag"
func (obs *Observations) AddFluxCsv(csvdir string, cxr map[int]int) {
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
			log.Fatalf("Observations.AddFluxCsv error: %v", err)
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
