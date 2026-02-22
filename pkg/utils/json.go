package utils

import "strings"

func CleanMarkdownJSON(s string) string {
	s = strings.TrimSpace(s)

	// Remove starting ``` or ```json
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimPrefix(s, "json")
		s = strings.TrimSpace(s)
	}

	// Remove ending ```
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}

	return s
}
