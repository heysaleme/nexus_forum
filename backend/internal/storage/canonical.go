package storage

import (
	"net/url"
	"strings"
)

// CanonicalMediaReference strips presigned query parameters and returns a stable object URL.
// External URLs (non-MinIO hosts) are returned unchanged.
func CanonicalMediaReference(reference string) string {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return ""
	}
	if strings.HasPrefix(reference, "/") {
		if i := strings.IndexAny(reference, "?#"); i >= 0 {
			return reference[:i]
		}
		return reference
	}
	u, err := url.Parse(reference)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return reference
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host, "/") + u.Path
}
