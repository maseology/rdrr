package basin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
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
