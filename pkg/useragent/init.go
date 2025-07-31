package useragent

import "sort"

func init() {
	// Ensure patterns are checked in specificity order to prevent false positives
	sort.Slice(browserPatterns, func(i, j int) bool {
		return browserPatterns[i].OrderHint < browserPatterns[j].OrderHint
	})
}
