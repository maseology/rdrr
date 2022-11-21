package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
)

func writeInts(fp string, i []int32) error {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, i); err != nil {
		return fmt.Errorf("WriteInts failed: %v", err)
	}
	if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		return fmt.Errorf("WriteInts failed: %v", err)
	}
	return nil
}

// func writeInt64s(fp string, i []int64) error {
// 	buf := new(bytes.Buffer)
// 	if err := binary.Write(buf, binary.LittleEndian, i); err != nil {
// 		return fmt.Errorf("WriteInts failed: %v", err)
// 	}
// 	if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
// 		return fmt.Errorf("WriteInts failed: %v", err)
// 	}
// 	return nil
// }

func writeFloats(fp string, f []float64) error {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, f); err != nil {
		return fmt.Errorf("WriteFloats failed: %v", err)
	}
	if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		return fmt.Errorf("WriteFloats failed: %v", err)
	}
	return nil
}

func write2Floats(fp string, f [][]float64) error {
	buf := new(bytes.Buffer)
	for _, v := range f {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			return fmt.Errorf("write2Floats failed: %v", err)
		}
	}
	return nil
	// buf := new(bytes.Buffer)
	// if err := binary.Write(buf, binary.LittleEndian, f); err != nil {
	// 	return fmt.Errorf("WriteFloats failed: %v", err)
	// }
	// if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
	// 	return fmt.Errorf("WriteFloats failed: %v", err)
	// }
	// return nil
}
