package postpro

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/maseology/mmio"
)

const jsonAPI = "https://api.oakridgeswater.ca/api/locnamsw?l="

type jdata struct {
	T string  `json:"Date"`
	V float64 `json:"Val"`
	F int32   `json:"RDTC"`
}

type ObsColl struct {
	T   []time.Time
	V   []float64
	nam string
}

func getJSON(url string) ([]time.Time, []float64, []int32, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 500 {
		return nil, nil, nil, nil
	} else if resp.StatusCode != http.StatusOK {
		return nil, nil, nil, fmt.Errorf("unexpected http GET status: %s", resp.Status)
	}

	var df []jdata
	err = json.NewDecoder(resp.Body).Decode(&df)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot decode JSON: %v", err)
	}

	dts, vals, flgs := make([]time.Time, len(df)), make([]float64, len(df)), make([]int32, len(df))
	for i, r := range df { // data queried is assumed to be pre-sorted
		t, err := time.Parse("2006-01-02T15:04:05", r.T)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("date parse error: %v", err)
		}
		dts[i] = t
		vals[i] = r.V
		flgs[i] = r.F
	}
	return dts, vals, flgs, nil
}

func saveGob(obsColls map[int]ObsColl, fp string) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return fmt.Errorf(" saveGob %v", err)
	}
	if err := gob.NewEncoder(f).Encode(obsColls); err != nil {
		return fmt.Errorf(" saveGob %v", err)
	}
	return nil
}

func loadGob(fp string) (map[int]ObsColl, error) {
	var obsColls map[int]ObsColl
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&obsColls)
	if err != nil {
		return nil, err
	}
	return obsColls, nil
}

func SaveObsToCsv(csvfp string) error {
	stanam := mmio.FileName(csvfp, false)
	dts, vals, flgs, err := getJSON(jsonAPI + stanam)
	if err != nil {
		return err
	}
	if dts == nil {
		return fmt.Errorf("%s: no data found", stanam)
	}
	_ = vals
	_ = flgs
	csvw := mmio.NewCSVwriter(csvfp)
	defer csvw.Close()
	csvw.WriteHead("Date,Flow,Flag")
	for i, t := range dts {
		csvw.WriteLine(t.Format("2006-01-02"), vals[i], flgs[i])
	}
	return nil
}

func GetObservations(odir, obsFP string) (map[int]ObsColl, error) {
	gg := func() (map[int]ObsColl, error) {
		f, err := os.Open(obsFP)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		recs := mmio.LoadCSV(io.Reader(f))
		obsColls := make(map[int]ObsColl, len(recs))
		for lns := range recs {
			staName := lns[0]
			cid, _ := strconv.Atoi(lns[1])
			fmt.Printf("%s (cid: %d): loading.. ", staName, cid)

			dts, vals, _, err := getJSON(jsonAPI + staName)
			if err != nil {
				return nil, err
			}
			if dts == nil {
				fmt.Println("no data found")
				continue
			}

			fmt.Printf("count = %d: %s to %s\n", len(dts), dts[0].Format("2006-01-02"), dts[len(dts)-1].Format("2006-01-02"))
			obsColls[cid] = ObsColl{dts, vals, staName}
		}
		return obsColls, nil
	}

	var c map[int]ObsColl
	var err error
	if _, ok := mmio.FileExists(odir + "obs.gob"); !ok {
		c, err := gg()
		if err != nil {
			return nil, err
		}
		saveGob(c, odir+"obs.gob")
	} else {
		c, err = loadGob(odir + "obs.gob")
		if err != nil {
			log.Fatalf(" getObservations loadGob failed: %v", err)
		}
	}
	return c, err
}
