package tem

// TEM topologic elevation model
type TEM struct {
	TECs map[int]*TEC
	us   map[int][]int
	c    int
}

// NumCells number of cells that make up the TEM
func (t *TEM) NumCells() int {
	return len(t.TECs)
}

// UnitContributingArea computes the (unit) contributing area from a given cell id
func (t *TEM) UnitContributingArea(cid int) float64 {
	t.c = 0
	t.climb(cid)
	return float64(t.c)
}

func (t *TEM) climb(cid int) {
	t.c++
	for _, i := range t.us[cid] {
		t.climb(i)
	}
}
