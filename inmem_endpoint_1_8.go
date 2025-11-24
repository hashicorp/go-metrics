// +build go1.8

package metrics

import (
	"sort"
)

// Sort the given PointValue slice in place using Go 1.8's sort.Slice.
func sortPoints(points []PointValue) {
	sort.Slice(points, func(i, j int) bool {
		return points[i].Name < points[j].Name
	})
}

// Sort the given GaugeValue slice in place using Go 1.8's sort.Slice.
func sortGauges(gauges []GaugeValue) {
	sort.Slice(gauges, func(i, j int) bool {
		return gauges[i].Hash < gauges[j].Hash
	})
}

// Sort the given SampledValue slice in place using Go 1.8's sort.Slice.
func sortSampled(sampled []SampledValue) {
	sort.Slice(sampled, func(i, j int) bool {
		return sampled[i].Hash < sampled[j].Hash
	})
}
