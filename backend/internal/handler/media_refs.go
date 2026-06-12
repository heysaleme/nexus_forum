package handler

import (
	"encoding/json"
	"strings"

	"nexus-forum-backend/internal/storage"
)

func canonicalizeMediaSlice(urls []string) []string {
	if len(urls) == 0 {
		return urls
	}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		if ref := storage.CanonicalMediaReference(u); ref != "" {
			out = append(out, ref)
		}
	}
	return out
}

func canonicalizeMediaJSON(mediaJSON string) string {
	mediaJSON = strings.TrimSpace(mediaJSON)
	if mediaJSON == "" || mediaJSON == "[]" || mediaJSON == "null" {
		return "[]"
	}
	var refs []string
	if err := json.Unmarshal([]byte(mediaJSON), &refs); err != nil {
		return mediaJSON
	}
	b, err := json.Marshal(canonicalizeMediaSlice(refs))
	if err != nil {
		return mediaJSON
	}
	return string(b)
}
