package handler

import "strings"

func isBase64DataURL(value string) bool {
	return strings.HasPrefix(value, "data:")
}

func rejectBase64MediaURLs(urls []string) bool {
	for _, u := range urls {
		if isBase64DataURL(u) {
			return true
		}
	}
	return false
}
