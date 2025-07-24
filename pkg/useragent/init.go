package useragent

import "sort"

func init() {
	// Sort patterns by OrderHint to ensure correct detection order
	sort.Slice(browserPatterns, func(i, j int) bool {
		return browserPatterns[i].OrderHint < browserPatterns[j].OrderHint
	})
}
