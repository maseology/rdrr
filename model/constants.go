package model

const (
	nearzero   = 1e-8
	fatalzero  = 1e-3
	steadyiter = 500
	secperday  = 86400.

	// cascade fraction parameters
	sill    = 1.
	nugget  = .001
	a       = .2     // scaling factor such that the "range" parameter looks representative (see fuzzy_slope.xlsx)
	gradMin = 0.0005 // smallest gradient from where lateral water movement is allowed

	strmkm2 = 1. // total drainage area [km²] required to deem a cell a "stream cell"

	avgRch = .1 / 366. // annual average groundwater recharge/initial groundwater discharge [m/day]
	// avgEp  = .6 / 366. // annual average potential evaporation [m/day]
	// minEp  = 0.        // baseline evaporation rate [m/day]
	// offset = -10       // offset to date of min Ep (adjusts the winter solstice 10 days before new years, i.e., Dec-21 'see sinET.xlsx)

	// twoThirds  = 2. / 3.
	// fiveThirds = 5. / 3.

	warmup = 365
)
