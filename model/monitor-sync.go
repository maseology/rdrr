package model

import (
	"sync"

	"github.com/maseology/mmio"
)

var gwg sync.WaitGroup
var gmu sync.Mutex
var tmu sync.Mutex

// var mondir string

// DeleteMonitors deletes monitor output from previous model run
func DeleteMonitors(mdldir string, preserveLast bool) {
	if preserveLast && mmio.DirExists(mdldir) {
		mmio.DeleteAllInDirectory(mdldir, ".last")
		for _, fp := range mmio.FileListExt(mdldir, ".cms") {
			mmio.MoveFile(fp, fp+".last")
		}
	}
	// mondir = mdldir
	mmio.MakeDir(mdldir)
	mmio.DeleteFile(mdldir + "g.yield.rmap")
	mmio.DeleteFile(mdldir + "g.ep.rmap")
	mmio.DeleteFile(mdldir + "g.aet.rmap")
	mmio.DeleteFile(mdldir + "g.olf.rmap")
	mmio.DeleteFile(mdldir + "g.ron.rmap")
	mmio.DeleteFile(mdldir + "g.rgen.rmap")
	mmio.DeleteFile(mdldir + "g.gwe.rmap")
	mmio.DeleteFile(mdldir + "g.sto.rmap")
	mmio.DeleteFile(mdldir + "g.sma.rmap")
	mmio.DeleteFile(mdldir + "g.Sdet.rmap")
	mmio.DeleteFile(mdldir + "g.wbal.rmap")
	mmio.DeleteAllInDirectory(mdldir, ".cms")
	mmio.DeleteAllInDirectory(mdldir, ".wbgt")
	// mmio.DeleteAllSubdirectories(mdldir)
}

// WaitMonitors waits for all writes to complete
func WaitMonitors() {
	gwg.Wait()
}
