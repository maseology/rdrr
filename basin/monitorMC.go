package basin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/maseology/mmio"
)

var mcdir string

// PrepMC creates a root Monte Carlo directory
func PrepMC(path string) {
	mcdir = path
	mmio.MakeDir(mcdir)
	rand.Seed(time.Now().UnixNano())
}

func setMCdir() {
	for {
		mondir = mcdir + fmt.Sprintf("%d/", rand.Uint32())
		if !mmio.IsDir(mondir) {
			break
		}
	}
	mmio.MakeDir(mondir)
}

type mcmonitor struct{ gy, ga, gr, gg, gb [][]float64 }

func (g *mcmonitor) print(pin map[int][]float64, xr map[int]int, ds []int, fnstep float64) {
	gwg.Add(1)
	gmu.Lock()
	defer gmu.Unlock()
	defer gwg.Done()
	nc := len(g.gy[0])
	my, ma, mr, mron, mg := make(map[int][]float32, nc), make(map[int][]float32, nc), make(map[int][]float32, nc), make(map[int][]float32, nc), make(map[int][]float32, nc)
	for c := range xr {
		my[c], ma[c], mr[c], mron[c], mg[c] = make([]float32, 12), make([]float32, 12), make([]float32, 12), make([]float32, 12), make([]float32, 12)
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
