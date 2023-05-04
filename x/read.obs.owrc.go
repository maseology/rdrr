package rdrr

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/maseology/mmio"
)

func OWRCstreamflow(odir string, dtb, dte time.Time) *Obs {
	fps, err := mmio.FileList(odir)
	if err != nil {
		log.Fatal(err)
	}

	croptodates := func(h hyd) hyd {
		m := make(map[int64]time.Time)
		for ut := range h {
			dt := time.Unix(ut, 0)
			if dtb.After(dt) || dte.Before(dt) {
				continue
			}
			m[ut] = dt
		}

		oo := make(hyd, len(m))
		for ut, dt := range m {
			oo[daydate(dt)] = h[ut]
		}
		return oo
	}

	if len(fps) > 0 {
		o := make(Obs, len(fps))
		for _, fp := range fps {
			cid, _ := strconv.Atoi(mmio.FileName(fp, false))
			o[cid] = loadHyd(fp)
		}
		return &o
	} else {
		fmt.Println(" gathering streamflow from api..")
		mh := get1001()
		o := make(Obs, len(mh))
		for cid, h := range mh {
			hc := croptodates(h)
			saveHyd(odir+strconv.Itoa(cid)+".gob", hc)
			o[cid] = hc
		}
		return &o
	}
}

// OWRC-API: collect timeseries data
func get1001() map[int]hyd {
	lns, _ := mmio.ReadTextLines("M:/OWRC-RDRR/owrc20-50-obs.csv") // to be made into API ///////////////////////////////////////////////////////////////
	o := make(map[int]hyd, len(lns[1:]))
	for _, ln := range lns[1:] {
		sp := strings.Split(ln, ",")
		lid, _ := strconv.Atoi(sp[0])
		cid, _ := strconv.Atoi(sp[4])

		url := "https://golang.oakridgeswater.ca/intgen/5/" + lidToIid(lid)
		fmt.Printf("    >> aquiring %s\n", url)

		o[cid] = readApiJson(url, 1001)
	}
	return o
}

// Misc. APIs
func readApiJson(url string, selRDNC int) hyd {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	type jdat struct {
		Date string  `json:"Date"`
		RDNC int     `json:"RDNC"`
		Val  float64 `json:"Val"`
		Unit int     `json:"unit"`
	}
	var d []jdat
	err = json.NewDecoder(resp.Body).Decode(&d)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(d)

	h := make(hyd)
	for _, dd := range d { // caution: coded assuming units are consistent /////////////////////////////////////////////////
		if dd.RDNC == selRDNC {
			tt, err := time.Parse(time.RFC3339, dd.Date)
			if err != nil {
				fmt.Println(err)
			}
			// h = append(h, hyd{tt, dd.Val})
			h[daydate(tt)] = dd.Val
		}
	}
	return h
}

func lidToIid(lid int) string {
	resp, err := http.Get("https://golang.oakridgeswater.ca/locinfo/" + strconv.Itoa(lid))
	if err != nil {
		log.Fatal(err)
	}

	type linfo struct {
		Iid  int    `json:"INT_ID"`
		Nam  string `json:"INT_NAME"`
		Nam2 string `json:"INT_NAME_ALT1"`
		Itc  int    `json:"INT_TYPE_CODE"`
		Lid  int    `json:"LOC_ID"`
	}
	var i []linfo
	err = json.NewDecoder(resp.Body).Decode(&i)
	if err != nil {
		log.Fatal(err)
	}
	return strconv.Itoa(i[0].Iid)
}
