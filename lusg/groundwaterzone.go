package lusg

// GWzoneColl holds a collection of GWzone.
type GWzoneColl map[int]GWzone

// GWzone holds model parameters and mapping associated with a groundwater zone
type GWzone struct {
	UCA           map[int]int // unit contributing areas per gw-zone: gwzid{cid{upcnt}}
	CidXR, StrmXR []int       // cross reference gw zones to cids/stream cells
}
