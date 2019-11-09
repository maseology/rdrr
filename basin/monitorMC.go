package basin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
)

var mcdir string

// var mcwg sync.WaitGroup

// PrepMC creates a root Monte Carlo directory
func PrepMC(path string) {
	mcdir = path
	mmio.MakeDir(mcdir)
	rand.Seed(time.Now().UnixNano())
}

// // WaitMC waits for all Monte Carlo writes to complete
// func WaitMC() {
// 	mcwg.Wait()
// }

func setMCdir() {
	for {
		mondir = mcdir + fmt.Sprintf("%d/", rand.Uint32())
		if !mmio.IsDir(mondir) {
			if _, ok := mmio.FileExists(mondir[:len(mondir)-1] + ".tar.gz"); !ok {
				break
			}
		}
	}
	mmio.MakeDir(mondir)
}

func compressMC(gd *grid.Definition) {
	// fmt.Println("reorg")
	// reorgMC(gd)
	// fmt.Println("compress")
	if err := mmio.CompressTarGZ(mondir[:len(mondir)-1]); err != nil {
		log.Fatalln("monitorMC.go compressMC() mmio.CompressTarGZ failed:", err)
	}
	mmio.DeleteFile(mondir) // didn't delete so that I can test tar.gz
}

type mcmonitor struct{ gy, ga, gr, gg, gb [][]float64 }

func (g *mcmonitor) print(pin map[int][]float64, xr map[int]int, ds []int, fnstep float64) {
	gwg.Add(1)
	gmu.Lock()
	defer gmu.Unlock()
	defer gwg.Done()
	nc := len(g.gy[0])
	my, ma, mr, mron, mg := make(map[int][]float32, nc), make(map[int][]float32, nc), make(map[int][]float32, nc), make(map[int][]float32, nc), make(map[int][]float32, nc)
	for c, i := range xr {
		my[c], ma[c], mr[c], mron[c], mg[c] = make([]float32, 12), make([]float32, 12), make([]float32, 12), make([]float32, 12), make([]float32, 12)
		if _, ok := mron[ds[i]]; !ok {
			mron[ds[i]] = make([]float32, 12)
		}
	}
	f := 30. * 1000. / fnstep
	for c, i := range xr {
		for mt := 0; mt < 12; mt++ {
			my[c][mt] = float32(g.gy[mt][i] * f)
			ma[c][mt] = float32(g.ga[mt][i] * f)
			mr[c][mt] = float32(g.gr[mt][i] * f)
			mg[c][mt] = float32((g.gg[mt][i] - g.gb[mt][i]) * f)
			if ds[i] > -1 {
				mron[ds[i]][mt] += float32(g.gr[mt][i] * f)
			}
			if _, ok := pin[i]; ok {
				for _, v := range pin[i] {
					mron[c][mt] += float32(v * f) // add inputs
				}
			}
		}
	}

	// NOTE: wbal = yield + ron - (aet + gwe + olf)
	writeRMAPmcmonitor(mondir+"g.yield.bin", my)
	writeRMAPmcmonitor(mondir+"g.aet.bin", ma)
	writeRMAPmcmonitor(mondir+"g.olf.bin", mr)
	writeRMAPmcmonitor(mondir+"g.ron.bin", mron)
	writeRMAPmcmonitor(mondir+"g.gwe.bin", mg)
}

func writeRMAPmcmonitor(filepath string, data map[int][]float32) error {
	buf := new(bytes.Buffer)
	for k, v := range data {
		if err := binary.Write(buf, binary.LittleEndian, int32(k)); err != nil {
			log.Fatalln("WriteBinary failed:", err)
		}
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			log.Fatalln("WriteBinary failed:", err)
		}
	}

	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

func reorgMC(gd *grid.Definition) {
	var wg sync.WaitGroup
	nc := gd.Ncells()
	for _, fp := range mmio.FileListExt(mondir, ".bin") {
		wg.Add(1)
		go func(fp string) {
			defer wg.Done()
			b, err := ioutil.ReadFile(fp)
			if err != nil {
				log.Fatalf("monitorMC.go reorgMC ioutil.ReadFile failed: %v", err)
			}
			buf := bytes.NewReader(b)

			type kv struct {
				k int32
				v [12]float32
			}
			coll := make([]kv, nc)
			for i := 0; i < nc; i++ {
				if err := binary.Read(buf, binary.LittleEndian, &coll[i].k); err != nil {
					if err == io.EOF {
						break
					}
					log.Fatalf("monitorMC.go reorgMC binary.Read failed: %v", err)
				}
				if err := binary.Read(buf, binary.LittleEndian, &coll[i].v); err != nil {
					log.Fatalf("monitorMC.go reorgMC binary.Read failed: %v", err)
				}
			}

			out := make([]float32, nc*12)
			for m := 0; m < 12; m++ {
				dat := make(map[int]float32, nc)
				for i := 0; i < nc; i++ {
					kv := coll[i]
					dat[int(kv.k)] = kv.v[m]
				}
				for i, c := range gd.Sactives {
					if v, ok := dat[c]; ok {
						out[m*nc+i] = v
					} //else {
					// 	log.Fatalf("monitorMC.go reorgMC error: inconsistent cell index: %d", c)
					// }
				}
			}
			if err := mmio.WriteBinary(mmio.RemoveExtension(fp)+".real", out); err != nil {
				log.Fatalf("monitorMC.go reorgMC WriteBinary failed: %v", err)
			}
			mmio.DeleteFile(fp)
		}(fp)
	}
	wg.Wait()
}
