package model

import (
	"encoding/json"
	"strings"
)

// MarshalJSON emits media_urls as a JSON array (single encoding).
// MediaUrls is stored in the DB as a JSON string; without this, presigned query
// parameters are double-encoded and clients may receive broken URLs.
func (p Post) MarshalJSON() ([]byte, error) {
	type Alias Post
	aux := struct {
		Alias
		MediaUrls []string `json:"media_urls"`
	}{
		Alias:     Alias(p),
		MediaUrls: []string{},
	}
	raw := strings.TrimSpace(p.MediaUrls)
	if raw != "" && raw != "[]" && raw != "null" {
		_ = json.Unmarshal([]byte(raw), &aux.MediaUrls)
	}
	if aux.MediaUrls == nil {
		aux.MediaUrls = []string{}
	}
	return json.Marshal(aux)
}
