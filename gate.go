package paintress

// FilterHighSeverity returns only HIGH severity d-mails from the input slice.
func FilterHighSeverity(dmails []DMail) []DMail {
	var high []DMail
	for _, dm := range dmails {
		if dm.Severity == "high" {
			high = append(high, dm)
		}
	}
	return high
}
