// Package chapters turns release chapter labels like "v.3 c.10-12" into numbers
// and computes how many chapters are new.
package chapters

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	volumeOrSeason = regexp.MustCompile(`\b(?:volume|vol|v|season|s)\.?\s*\d+(?:\.\d+)?`)
	number         = regexp.MustCompile(`\d+(?:\.\d+)?`)
)

// Parse returns the highest chapter number in raw, ignoring volume and season
// numbers. ok is false when raw contains no chapter number.
func Parse(raw string) (float64, bool) {
	s := volumeOrSeason.ReplaceAllString(strings.ToLower(raw), " ")
	best, found := 0.0, false
	for _, m := range number.FindAllString(s, -1) {
		n, err := strconv.ParseFloat(m, 64)
		if err != nil {
			continue
		}
		if !found || n > best {
			best, found = n, true
		}
	}
	return best, found
}

// NewCount is the number of whole chapters after mine up to latest; never negative.
func NewCount(latest, mine float64) int {
	if latest <= 0 {
		return 0
	}
	n := int(math.Floor(latest) - math.Floor(mine))
	if n < 0 {
		return 0
	}
	return n
}

// Effective is the latest chapter to show: the fetched value, or the manual
// override when it is higher.
func Effective(fetched float64, manual *float64) float64 {
	if manual != nil && *manual > fetched {
		return *manual
	}
	return fetched
}
