package rdrr

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
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

func writeFloats32(gd *grid.Definition, fp string, f []float64) error {
	f32 := func() []float32 { // ** due to round-off error, outputting as float32 causes waterbalance issues **
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
	if gd != nil {
		if err := gd.ToHDRfloat(mmio.RemoveExtension(fp)+".hdr", 1, 32); err != nil {
			return fmt.Errorf("writeInts (hdr) failed: %v", err)
		}
	}
	return nil
}

func writeInts(gd *grid.Definition, fp string, i []int32) error {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, i); err != nil {
		return fmt.Errorf("writeInts failed: %v", err)
	}
	if err := os.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
		return fmt.Errorf("writeInts failed: %v", err)
	}
	if err := gd.ToHDR(mmio.RemoveExtension(fp)+".hdr", 1, 32); err != nil {
		return fmt.Errorf("writeInts (hdr) failed: %v", err)
	}
	return nil
}

// writes float32 map of user define stream flow monitoring to a Go binary (*.gob)
func writeMons(fp string, swsmons [][]int, qs [][]float64) error {
	f32 := func(f []float64) []float32 {
		o := make([]float32, len(f))
		for i, v := range f {
			o[i] = float32(v)
		}
		return o
	}
	m32 := func() map[int][]float32 {
		o := make(map[int][]float32)
		n := 0
		for _, cs := range swsmons {
			for _, c := range cs {
				o[c] = f32(qs[n])
				n++
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
