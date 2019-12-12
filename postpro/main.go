package main

import (
	"encoding/binary"
	"log"
	"math"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
)

func main() {
	tt := mmio.NewTimer()
	type coll struct {
		K int32
		V [12]float32
	}
	gd, err := grid.ReadGDEF("E:/ormgp_rdrr/ORMGP_50_hydrocorrect.uhdem.gdef", true)
	if err != nil {
		log.Fatalf("Fatal error: read failed: %v", err)
	}
	buf := mmio.OpenBinary("C:/Users/Mason/Desktop/g.yield.bin")
	uc := make([]coll, gd.Nactives())
	if err := binary.Read(buf, binary.LittleEndian, uc); err != nil {
		log.Fatalf("Fatal error: read failed: %v", err)
	}

	lout := make([]int16, gd.Ncells()*24)
	for i := 0; i < gd.Ncells()*24; i++ {
		lout[i] = math.MinInt16 //-9999
	}
	for _, c := range uc {
		cid := int(c.K)
		for i := 0; i < 12; i++ {
			v := c.V[i] * 12.
			var iv int16
			if v < float32(math.MinInt16+1) {
				iv = math.MinInt16 + 1
			} else if v > float32(math.MaxInt16) {
				iv = math.MaxInt16
			} else {
				iv = int16(v)
			}
			// lout[cid+2*i*gd.Ncells()] = iv - 1
			// lout[cid+(2*i+1)*gd.Ncells()] = iv + 1
			lout[cid*24+2*i] = iv - 1
			lout[cid*24+(2*i+1)] = iv + 1
		}
	}
	mmio.WriteBinary("C:/Users/Mason/Desktop/g.yield.bip", lout)
	if err := gd.ToHDR("C:/Users/Mason/Desktop/g.yield.hdr", 24); err != nil {
		log.Fatalf("Fatal error: gd.ToHDR failed: %v", err)
	}
	tt.Print("complete")
}
