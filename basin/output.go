package basin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
)

// saveRMap is used to output grid data as key-value pairs (*.rmap)
// meant for singular data (i.e., long-term annual average)
func saveRMap(d map[int]float64, fp string) {
	buf := new(bytes.Buffer)
	for k, v := range d {
		if err := binary.Write(buf, binary.LittleEndian, int32(k)); err != nil {
			fmt.Println("saveRMap key write failed:", err)
		}
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			fmt.Println("saveRMap value write failed:", err)
		}
	}
	if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		fmt.Printf("ioutil.WriteFile failed: %v\n", err)
	}
}

// saveIMap is used to output grid data as key-value pairs (*.imap)
// meant for singular data (i.e., long-term annual average)
func saveIMap(d map[int]int, fp string) {
	buf := new(bytes.Buffer)
	for k, v := range d {
		if err := binary.Write(buf, binary.LittleEndian, int32(k)); err != nil {
			log.Fatalf("saveIMap key write failed: %v\n %s", err, fp)
		}
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			log.Fatalf("saveIMap value write failed: %v\n %s", err, fp)
		}
	}
	if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		log.Fatalf("ioutil.WriteFile failed: %v\n %s", err, fp)
	}
}
