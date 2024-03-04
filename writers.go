package rdrr

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"os"
)

// writes to float32
func writeFloats(fp string, f []float64) error {
	f32 := func() []float32 {
		o := make([]float32, len(f))
		for i, v := range f {
			o[i] = float32(v)
		}
		return o
	}()
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, f32); err != nil {
		return fmt.Errorf("writeFloats failed: %v", err)
	}
	if err := os.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		return fmt.Errorf("writeFloats failed: %v", err)
	}
	return nil
}

func writeInts(fp string, i []int32) error {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, i); err != nil {
		return fmt.Errorf("writeInts failed: %v", err)
	}
	if err := os.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		return fmt.Errorf("writeInts failed: %v", err)
	}
	return nil
}

// writes float32 map to a gob
func writeMons(fp string, swsmons []int, qs [][]float64) error {
	f32 := func(f []float64) []float32 {
		o := make([]float32, len(f))
		for i, v := range f {
			o[i] = float32(v)
		}
		return o
	}
	m32 := func() map[int][]float32 {
		o := make(map[int][]float32)
		for k, c := range swsmons {
			if c >= 0 {
				o[c] = f32(qs[k])
			}

		}
		return o
	}()
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	err = enc.Encode(m32)
	if err != nil {
		return err
	}
	return nil
}
