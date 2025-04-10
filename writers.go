package rdrr

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"os"
)

// writes to binary
func writeFloats(fp string, f []float64) error {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, f); err != nil {
		return fmt.Errorf("writeFloats failed: %v", err)
	}
	if err := os.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		return fmt.Errorf("writeFloats failed: %v", err)
	}
	return nil
}

// writes float32 map of user define stream flow monitoring to a Go binary (*.gob)
func writeMons(fp string, q []float64) error {
	f32 := func() []float32 {
		o := make([]float32, len(q))
		for i, v := range q {
			o[i] = float32(v)
		}
		return o
	}()
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	err = enc.Encode(f32)
	if err != nil {
		return err
	}
	return nil
}
