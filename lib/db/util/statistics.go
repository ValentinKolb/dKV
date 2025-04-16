// Package util provides testing, benchmarking, and utility tools for KVDB implementations.
// This file implements a specialized size histogram for efficient tracking and analysis
// of data size distributions. The histogram uses exponential bucket sizing to cover
// a wide range of values (bytes to gigabytes) with minimal memory overhead.
//
// Key features include:
//   - Efficient memory usage through bucketing
//   - Thread-safe sample addition and querying
//   - Statistical estimators (median, percentiles)
//   - Distribution analysis capabilities
//
// This utility is particularly useful for database implementations that need to
// report on data characteristics without performing expensive full scans.
package util

import (
	"math"
	"sync"
)

// ----------------------------------------------------------------------------
// Helper functions
// ----------------------------------------------------------------------------

type Stats struct {
	StdDeviation float64 `json:"std_deviation"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	Mean         float64 `json:"mean"`
	MinMaxRatio  float64 `json:"min_max_ratio"`
}

// NewStats computes the standard deviation, minimum, and maximum values
// from an array of float64 values.
func NewStats(values []float64) Stats {
	if len(values) == 0 {
		return Stats{}
	}

	// initialize min and max with the first value
	min := values[0]
	max := values[0]

	// calculate sum for mean
	var sum float64
	for _, v := range values {
		sum += v

		// update min and max while iterating
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// calculate mean
	mean := sum / float64(len(values))

	// calculate sum of squared differences from mean
	var sumSquaredDiffs float64
	for _, v := range values {
		diff := v - mean
		sumSquaredDiffs += diff * diff
	}

	// calculate standard deviation (population formula)
	stdDev := math.Sqrt(sumSquaredDiffs / float64(len(values)))

	// calculate min/max ratio
	var minMaxRatio float64 = 1.0
	if max > 0 {
		minMaxRatio = min / max
	}

	return Stats{
		StdDeviation: stdDev,
		Min:          min,
		Max:          max,
		Mean:         mean,
		MinMaxRatio:  minMaxRatio,
	}
}

type DistributionStats struct {
	Stats
	DistributionQuality float64 `json:"distribution_quality"`
}

// NewDistributionStats computes quality metrics for value distribution
func NewDistributionStats(shardSizes []float64) DistributionStats {
	// get statistics
	stats := NewStats(shardSizes)

	// calculate coefficient of variation
	var cv float64
	if stats.Mean > 0 {
		cv = stats.StdDeviation / stats.Mean
	}

	// distribution quality combines CV and min/max ratio
	// -> lower CV and higher min/max ratio indicate better distribution
	distributionQuality := (1.0-math.Min(1.0, cv))*0.5 + stats.MinMaxRatio*0.5

	return DistributionStats{
		Stats:               stats,
		DistributionQuality: distributionQuality,
	}
}

// ----------------------------------------------------------------------------
// SizeHistogram
// ----------------------------------------------------------------------------

// SizeHistogram tracks the distribution of data sizes
// It organizes sizes into buckets for efficient memory usage
// while still providing accurate size estimations.
// Supports tracking values from bytes to multiple gigabytes.
type SizeHistogram struct {
	mutex      sync.RWMutex
	boundaries []int   // Bucket boundaries covering byte to GB range
	buckets    []int64 // Count of items in each bucket
	count      int64   // Total number of samples
	sum        int64   // Sum of all sampled sizes
}

// NewSizeHistogram creates a new size histogram with default bucket boundaries
// The boundaries are calibrated to handle sizes from bytes to gigabytes
func NewSizeHistogram() *SizeHistogram {
	// Using exponential bucket sizes to cover a wide range efficiently
	// This covers from small values to over 4GB
	return &SizeHistogram{
		boundaries: []int{
			16, 64, 256, 1024, 4096, // Bytes: 16B to 4KB
			16384, 65536, 262144, 1048576, // KB range: 16KB to 1MB
			4194304, 16777216, 67108864, // MB range: 4MB to 64MB
			268435456, 1073741824, 4294967296, // Above 256MB to 4GB
		},
		buckets: make([]int64, 16), // 16 buckets (15 boundaries + 1 for larger values)
	}
}

// AddSample adds a size sample to the histogram
//
// Thread-safe: This method is safe for concurrent use
func (h *SizeHistogram) AddSample(size int) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Find the appropriate bucket for this size
	bucketIndex := 0
	for i, boundary := range h.boundaries {
		if size <= boundary {
			bucketIndex = i
			break
		}
		bucketIndex = len(h.boundaries) // Last bucket for all larger values
	}

	// Update statistics
	h.buckets[bucketIndex]++
	h.count++
	h.sum += int64(size)
}

// GetCount returns the total number of samples
//
// Thread-safe: This method is safe for concurrent use
func (h *SizeHistogram) GetCount() int64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.count
}

// AverageSize returns the average size across all samples
//
// Thread-safe: This method is safe for concurrent use
func (h *SizeHistogram) AverageSize() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.count == 0 {
		return 0
	}
	return int(h.sum / h.count)
}

// MedianEstimate estimates the median size based on the histogram
//
// Thread-safe: This method is safe for concurrent use
func (h *SizeHistogram) MedianEstimate() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.count == 0 {
		return 0
	}

	// Find the median bucket
	medianCount := h.count / 2
	cumulativeCount := int64(0)

	for i, count := range h.buckets {
		cumulativeCount += count
		if cumulativeCount >= medianCount {
			// Found the median bucket
			if i == 0 {
				return h.boundaries[0] / 2
			} else if i < len(h.boundaries) {
				return (h.boundaries[i-1] + h.boundaries[i]) / 2
			} else {
				// Estimation for the last bucket (2x the last boundary)
				return h.boundaries[len(h.boundaries)-1] * 2
			}
		}
	}

	// Shouldn't happen but as a fallback
	return int(h.sum / h.count)
}

// Reset clears all histogram data
//
// Thread-safe: This method is safe for concurrent use
func (h *SizeHistogram) Reset() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.count = 0
	h.sum = 0
	for i := range h.buckets {
		h.buckets[i] = 0
	}
}

// GetPercentileEstimate returns an estimate for the given percentile (0-100)
//
// Thread-safe: This method is safe for concurrent use
func (h *SizeHistogram) GetPercentileEstimate(percentile int) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.count == 0 || percentile < 0 || percentile > 100 {
		return 0
	}

	// Calculate target count for percentile
	targetCount := int64(math.Ceil(float64(h.count) * float64(percentile) / 100.0))
	cumulativeCount := int64(0)

	for i, count := range h.buckets {
		cumulativeCount += count
		if cumulativeCount >= targetCount {
			// Found the percentile bucket
			if i == 0 {
				// For the first bucket, estimate as half of the boundary
				return h.boundaries[0] / 2
			} else if i < len(h.boundaries) {
				// For middle buckets, use the average of boundaries
				return (h.boundaries[i-1] + h.boundaries[i]) / 2
			} else {
				// For the last bucket, estimate as 2x the last boundary
				return h.boundaries[len(h.boundaries)-1] * 2
			}
		}
	}

	// Should never reach here
	return int(h.sum / h.count)
}

// SizeDistribution returns the distribution of samples across buckets
// Returns two slices: bucket boundaries and the percentage in each bucket
//
// Thread-safe: This method is safe for concurrent use
func (h *SizeHistogram) SizeDistribution() ([]int, []float64) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.count == 0 {
		return h.boundaries, make([]float64, len(h.buckets))
	}

	// Calculate percentages
	percentages := make([]float64, len(h.buckets))
	for i, count := range h.buckets {
		percentages[i] = float64(count) * 100.0 / float64(h.count)
	}

	return h.boundaries, percentages
}
