package basin

import "github.com/maseology/mmio"

func sumPlotSto(fp string, h, d []float64) {
	xs, ys := make([]float64, len(h)), make(map[string][]float64, 2)
	for i := range h {
		xs[i] = float64(i)
	}
	ys["sto"] = h
	ys["def"] = d
	mmio.Line(fp, xs, ys)
}

// func sumPlotHydrographWB(fp string, s, d, k, x, a, g []interface{}) {
// 	xs, ys := make([]float64, len(s)), make(map[string][]float64, 6)
// 	for i := range s {
// 		xs[i] = float64(i)
// 	}
// 	ys["sto"] = mmio.InterfaceToFloat(s)
// 	ys["def"] = mmio.InterfaceToFloat(d)
// 	ys["lag"] = mmio.InterfaceToFloat(k)
// 	ys["xs"] = mmio.InterfaceToFloat(x)
// 	ys["aet"] = mmio.InterfaceToFloat(a)
// 	ys["rch"] = mmio.InterfaceToFloat(g)
// 	mmio.Line(fp, xs, ys)
// }
