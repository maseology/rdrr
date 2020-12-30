package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/maseology/mmio"
)

type data struct {
	T string  `json:"Date"`
	V float64 `json:"Val"`
	F int32   `json:"RDTC"`
}

type coll struct {
	T   []time.Time
	V   []float64
	nam string
}

func getJSON(url string) ([]time.Time, []float64, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 500 {
		return nil, nil, nil
	} else if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected http GET status: %s", resp.Status)
	}

	var df []data
	err = json.NewDecoder(resp.Body).Decode(&df)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot decode JSON: %v", err)
	}

	dts, vals := make([]time.Time, len(df)), make([]float64, len(df))
	for i, r := range df { // data queried is assumed to be pre-sorted
		t, err := time.Parse("2006-01-02T15:04:05", r.T)
		if err != nil {
			return nil, nil, fmt.Errorf("date parse error: %v", err)
		}
		dts[i] = t
		vals[i] = r.V
	}
	return dts, vals, nil
}

func saveGob(colls map[int]coll, fp string) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return fmt.Errorf(" saveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(colls); err != nil {
		return fmt.Errorf(" saveGob %v", err)
	}
	return nil
}

func loadGob(fp string) (map[int]coll, error) {
	var colls map[int]coll
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&colls)
	if err != nil {
		return nil, err
	}
	return colls, nil
}

func getObservations(stationCSVfp string) (map[int]coll, error) {

	f, err := os.Open(obsFP)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	recs := mmio.LoadCSV(io.Reader(f))
	colls := make(map[int]coll, len(recs))
	for lns := range recs {
		staName := lns[0]
		cid, _ := strconv.Atoi(lns[1])
		fmt.Printf("%s (cid: %d): loading.. ", staName, cid)

		dts, vals, err := getJSON(jsonAPI + staName)
		if err != nil {
			return nil, err
		}
		if dts == nil {
			fmt.Println("no data found")
			continue
		}

		fmt.Printf("count = %d: %s to %s\n", len(dts), dts[0].Format("2006-01-02"), dts[len(dts)-1].Format("2006-01-02"))
		colls[cid] = coll{dts, vals, staName}
	}
	return colls, nil
}
