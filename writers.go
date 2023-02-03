package rdrr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
)

func writeFloats(fp string, f []float64) error {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, f); err != nil {
		return fmt.Errorf("writeFloats failed: %v", err)
	}
	if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		return fmt.Errorf("writeFloats failed: %v", err)
	}
	return nil
}

func writeInts(fp string, i []int32) error {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, i); err != nil {
		return fmt.Errorf("writeInts failed: %v", err)
	}
	if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		return fmt.Errorf("writeInts failed: %v", err)
	}
	return nil
}
