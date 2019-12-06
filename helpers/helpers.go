package helpers

// StringArrayContains returns true of a value is present in the array
func StringArrayContains(array []string, value string) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}
