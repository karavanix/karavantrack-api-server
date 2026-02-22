package utils

func Truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func TruncateWithSuffix(s string, n int, suffix string) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + suffix
}

func TruncateWithPrefix(s string, n int, prefix string) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return prefix + string(runes[n:])
}
