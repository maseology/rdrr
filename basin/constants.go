package basin

const (
	nearzero   = 1e-8
	steadyiter = 500
	secperday  = 86400.
	minslope   = 0.001
	strmkm2    = 1. // total drainage area [km²] required to deem a cell a "stream cell"

	avgEp  = .6 / 366. // average annual potential evaporation [m/day]
	minEp  = 0.        // baseline evaporation rate [m/day]
	offset = -10       // offset to date of min Ep (adjusts the winter solstice 10 days before new years, i.e., Dec-21 'see sinET.xlsx)

	twoThirds  = 2. / 3.
	fiveThirds = 5. / 3.
)
