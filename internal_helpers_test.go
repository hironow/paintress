package paintress

// containsStr is a test helper -- checks substring presence without importing strings.
// This duplicate exists for package paintress (internal) tests; the package
// paintress_test version lives in helpers_test.go.
func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
