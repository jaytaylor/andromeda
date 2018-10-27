package contains

// String returns true if the sequence of items contains value s.
func String(items []string, s string) bool {
	for _, item := range items {
		if item == s {
			return true
		}
	}
	return false
}
