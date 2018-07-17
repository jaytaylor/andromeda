package unique

import (
	"sort"
)

// Strings returns a unique subset of the string slice provided.
//
// Also sorts the result.
func Strings(input []string) []string {
	u := make([]string, 0, len(input))
	m := map[string]struct{}{}
	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = struct{}{}
			u = append(u, val)
		}
	}
	return u
}

// StringsSorted sorts the result before returning it.
func StringsSorted(input []string) []string {
	u := Strings(input)
	sort.Strings(u)
	return u
}
