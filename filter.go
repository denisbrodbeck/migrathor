package migrathor

import "strings"

// filterExcept returns a new slice containing all strings
// which exist in the first but not the second slice.
func filterExcept(items []string, except []string) []string {
	return filter(items, func(s string) bool {
		for _, ex := range except {
			if strings.ToLower(s) == strings.ToLower(ex) {
				return false
			}
		}
		return true
	})
}

// filter returns a new slice containing all strings in the slice that satisfy the predicate f.
func filter(items []string, f func(string) bool) []string {
	filtered := []string{}
	for _, v := range items {
		if f(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}
