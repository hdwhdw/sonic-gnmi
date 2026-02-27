package integration

// Concat joins two strings with a separator.
func Concat(a, sep, b string) string {
	return a + sep + b
}

// Repeat returns s repeated n times.
func Repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// Truncate shortens s to at most maxLen characters.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// CountChar returns how many times ch appears in s.
func CountChar(s string, ch byte) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			count++
		}
	}
	return count
}
