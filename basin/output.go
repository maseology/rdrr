package basin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"

	"github.com/maseology/goHydro/hru"
)

// saveBinaryMap1 is used to output grid data as key-value pairs (*.rmap)
// meant for singular data (i.e., long-term annual average)
func saveBinaryMap1(d map[int]float64, fp string) {
	buf := new(bytes.Buffer)
	for k, v := range d {
		if err := binary.Write(buf, binary.LittleEndian, int32(k)); err != nil {
			fmt.Println("saveBinaryMap1 key write failed:", err)
		}
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			fmt.Println("saveBinaryMap1 value write failed:", err)
		}
	}
	if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		fmt.Printf("ioutil.WriteFile failed: %v\n", err)
	}
}

func printHRUprops(ws hru.WtrShd) {
	perc, fimp, cap := make(map[int]float64, len(ws)), make(map[int]float64, len(ws)), make(map[int]float64, len(ws))
	for i, h := range ws {
		perc[i], fimp[i], cap[i] = h.PercFimpCap()
		cap[i] *= 1000. // [m] to [mm]
	}
	saveBinaryMap1(perc, "hru.perc_mpts.rmap")
	saveBinaryMap1(fimp, "hru.fimp.rmap")
	saveBinaryMap1(cap, "hru.cap_mm.rmap")
}
