package main

import (
	"encoding/gob"
	"os"
)

func saveGOBi(fp string, d []int) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(d)
	if err != nil {
		return err
	}
	return nil
}

func loadGOBi(fp string) ([]int, error) {
	var d []int
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&d)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func saveGOBa(fp string, d []map[int][]float64) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(d)
	if err != nil {
		return err
	}
	return nil
}

func loadGOBa(fp string) ([]map[int][]float64, error) {
	var d []map[int][]float64
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&d)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func saveGOB(fp string, d []map[int]float64) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(d)
	if err != nil {
		return err
	}
	return nil
}

func loadGOB(fp string) ([]map[int]float64, error) {
	var d []map[int]float64
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&d)
	if err != nil {
		return nil, err
	}
	return d, nil
}
