package helpers

func BoolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
