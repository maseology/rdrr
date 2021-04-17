package model

import (
	"fmt"

	"github.com/maseology/mmio"
)

func printSample(b *subdomain, s *sample, chkdir string) {
	if len(chkdir) > 0 {
		mmio.MakeDir(chkdir)
		b.write(chkdir)
		s.write(chkdir)
		// os.Exit(2)
	}
	mmio.FileRename("hyd.png", "hyd.last.png", true)
	fmt.Printf(" number of subwatersheds: %d\n", len(s.gw))
	fmt.Printf("\n running model..\n\n")
}
