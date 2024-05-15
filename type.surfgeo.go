package rdrr

import "log"

// surficial geology types
const (
	Impermeable = iota

	// 1-5 relative permeability
	Low
	Low_Medium
	Medium
	Medium_High
	High

	// 6-8
	Variable
	Streambed
	Wetland_Sediments

	BedrockWithDrift
)

// KsatFromID returns an approximate estimate of saturated hydraulic conductivity [m/s] for a given material type
func KsatFromID(sgid int) float64 {
	switch sgid {
	case Low, BedrockWithDrift:
		return 1e-9
	case Low_Medium:
		return 1e-8 // ~316 mm/yr
	case Medium:
		return 1e-7
	case Medium_High:
		return 1e-6
	case High:
		return 1e-5
	case Variable: // (unknown)
		return 1e-8
	case Streambed: // (alluvium/unconsolidated/fluvial/floodplain)
		return 1e-5
	case Wetland_Sediments: // (organics)
		return 1e-6
	default:
		log.Fatalf("rdrr.KsatFromID: no permeability assigned to material type ID %d", sgid)
		return 0.
	}
}
