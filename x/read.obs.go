package rdrr

// type itime int64 // hashing with time.Time is not desirable

type Obs map[int]map[int64]float64 // monitor id; (i64/unix-)date-value
