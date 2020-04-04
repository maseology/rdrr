package basin

import (
	"fmt"
	"log"
	"strings"

	"github.com/maseology/mmio"
)

// RunDefault runs simulation with default parameters
func RunDefault(mdldir, metfp, chkdir string, topm, smax, dinc, soildepth, kfact float64, print bool) float64 {
	tt := mmio.NewTimer()
	b, _ := masterToSubomain(metfp)
	b.mdldir = mdldir

	if print {
		tt.Lap("sub-domain load complete")
		fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))
		fmt.Printf(" building sample HRUs and TOPMODEL\n")
	}
	smpl := b.toDefaultSample(topm, smax, soildepth, kfact)

	if print {
		tt.Lap("sample build complete")
		if len(chkdir) > 0 {
			mmio.MakeDir(chkdir)
			b.write(chkdir)
			smpl.print(chkdir)
			tt.Lap("sample maps printed")
		}
		mmio.FileRename("hyd.png", "hydx.png", true)
		fmt.Printf(" number of subwatersheds: %d\n", len(smpl.gw))
		fmt.Printf("\n running model..\n\n")
	}

	dt, y, ep, obs, intvl, nstep := b.getForcings()
	return b.eval(&smpl, dt, y, ep, obs, intvl, nstep, dinc, topm, print)
}

// RunMaster runs simulation of the entire masterdomain with default parameters
func RunMaster(mdldir, metfp, chkdir string, topm, smax, dinc, soildepth, kfact float64, print bool) float64 {
	tt := mmio.NewTimer()
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunMaster error: masterDomain is empty\n")
	}
	if print {
		fmt.Println("Building Sub Domain..")
	}
	var b subdomain
	var frc *FORC
	if len(metfp) == 0 {
		if masterDomain.frc == nil {
			log.Fatalf(" basin.RunMaster error: no forcings made available\n")
		}
		frc, _ = masterForcing()
	} else if strings.ToLower(metfp) == "gob" {
		frc, _ = loadGOBforcing(mdldir+"met/", print)
	} else {
		frc, _ = loadForcing(metfp, print)
	}
	b = masterDomain.noSubDomain(frc)
	b.mdldir = mdldir
	b.cid0 = -1
	if len(b.rtr.swscidxr) == 1 {
		b.rtr.swscidxr = map[int][]int{-1: b.cids}
		b.rtr.sws = make(map[int]int, b.ncid)
		for _, c := range b.cids {
			b.rtr.sws[c] = -1
		}
	}

	if print {
		tt.Lap("domain load complete")
		fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))
		fmt.Printf(" building sample HRUs and TOPMODEL.. ")
	}
	smpl := b.toDefaultSample(topm, smax, soildepth, kfact)

	if print {
		tt.Lap("complete")
		if len(chkdir) > 0 {
			mmio.MakeDir(chkdir)
			// masterDomain.gd.SaveAs(chkdir + "masterDomain.gdef")
			b.write(chkdir)
			smpl.print(chkdir)
			tt.Lap("sample maps printed")
		}
		mmio.FileRename("hyd.png", "hydx.png", true)
		fmt.Printf(" number of subwatersheds: %d\n", len(smpl.gw))
		fmt.Printf("\n running model..\n\n")
	}

	dt, y, ep, obs, intvl, nstep := b.getForcings()
	return b.eval(&smpl, dt, y, ep, obs, intvl, nstep, dinc, topm, print)
}
