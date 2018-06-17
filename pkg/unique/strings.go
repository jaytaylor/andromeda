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
	sort.Strings(u)
	return u
}
