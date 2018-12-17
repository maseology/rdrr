package grid

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/maseology/mmio"
)

// ReadGDEF imports a grid definition file
func ReadGDEF(fp string) {

	a, stErr, uni := mmio.ReadTextLines(fp), make([]string, 0), false
	errfunc := func(v string, err error) {
		stErr = append(stErr, fmt.Sprintf("     failed to read '%v': %v", v, err))
	}

	oe, err := strconv.ParseFloat(a[0], 64)
	if err != nil {
		errfunc("OE", err)
	}
	on, err := strconv.ParseFloat(a[1], 64)
	if err != nil {
		errfunc("ON", err)
	}
	rot, err := strconv.ParseFloat(a[2], 64)
	if err != nil {
		errfunc("ROT", err)
	}
	nr, err := strconv.ParseInt(a[3], 10, 32)
	if err != nil {
		errfunc("NR", err)
	}
	nc, err := strconv.ParseInt(a[4], 10, 32)
	if err != nil {
		errfunc("NC", err)
	}
	cs, err := strconv.ParseFloat(a[5], 64)
	if err != nil {
		if a[5][0] == 85 { // 85 = acsii code for 'U'
			uni = true
		} else {
			errfunc("CS", err)
		}
		cs, err = strconv.ParseFloat(a[5][1:len(a[5])], 64)
		if err != nil {
			errfunc("NC", err)
		}
	} else {
		log.Fatalf(" *** Fatal error: ReadGDEF: non-uniform grids currently not supported")
	}

	// error checking
	if len(stErr) > 0 {
		fmt.Println(" *** Fatal error(s): ReadGDEF ***")
		for _, v := range stErr {
			fmt.Println(v)
		}
		log.Fatalf(" ***")
	}

	fmt.Println(oe)
	fmt.Println(on)
	fmt.Println(rot)
	fmt.Println(nr)
	fmt.Println(nc)
	fmt.Println(cs)
	fmt.Println(uni)

	if len(a) > 6 { // active cells
		fmt.Println(len(a[6]))
		b := []byte(a[6]) // byte array
		for i, v := range b {
			s := fmt.Sprintf("%b", v)
			fmt.Printf("%s%s", strings.Repeat("0", 8-len(s)), s)
			if (i+1)%int(nc) == 0 {
				println()
				break
			}
			// break
		}
	}

}
