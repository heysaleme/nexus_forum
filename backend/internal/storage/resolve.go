package storage

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// DefaultPresignExpiry is how long presigned MinIO URLs remain valid.
const DefaultPresignExpiry = 24 * time.Hour

// ObjectStoreWithAccess extends ObjectStore with URL resolution for private buckets.
type ObjectStoreWithAccess interface {
	ObjectStore
	AccessibleURL(ctx context.Context, reference string) (string, error)
}

// ResolveMediaJSON replaces stored media references with browser-accessible URLs.
func ResolveMediaJSON(ctx context.Context, store ObjectStore, mediaJSON string) string {
	mediaJSON = strings.TrimSpace(mediaJSON)
	if mediaJSON == "" || mediaJSON == "[]" || mediaJSON == "null" {
		return "[]"
	}
	var refs []string
	if err := json.Unmarshal([]byte(mediaJSON), &refs); err != nil {
		return mediaJSON
	}
	accessor, ok := store.(ObjectStoreWithAccess)
	if !ok {
		return mediaJSON
	}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		resolved, err := accessor.AccessibleURL(ctx, ref)
		if err != nil || resolved == "" {
			out = append(out, ref)
			continue
		}
		out = append(out, resolved)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return mediaJSON
	}
	return string(b)
}
