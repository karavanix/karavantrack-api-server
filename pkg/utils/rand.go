package utils

import (
	"math/rand"
)

// RandomInt returns a single random integer in [min, max] inclusive.
// If min > max, they are swapped.
func RandomInt(min, max int) int {
	if min > max {
		min, max = max, min
	}
	return rand.Intn(max-min+1) + min
}

// RandInts returns `count` random ints in [min, max] inclusive using the
// global generator (auto-seeded since Go 1.20). Returns error if count < 1.
func RandomInts(min, max, count int) []int {
	out := make([]int, count)

	if count < 1 {
		return out
	}

	if min > max {
		min, max = max, min
	}
	size := max - min + 1

	for i := range count {
		out[i] = rand.Intn(size) + min
	}
	return out
}

// RandomDistinctInts returns `count` distinct random ints in [min, max] inclusive.
// Returns empty slice if count < 1 or count > range size.
func RandomDistinctInts(min, max, count int) []int {
	if count < 1 || min > max {
		return []int{}
	}

	size := max - min + 1
	if count > size {
		// cannot pick more distinct numbers than available in range
		return []int{}
	}

	// make a slice of all possible numbers
	nums := make([]int, size)
	for i := range size {
		nums[i] = min + i
	}

	// shuffle
	rand.Shuffle(size, func(i, j int) {
		nums[i], nums[j] = nums[j], nums[i]
	})

	return nums[:count]
}
