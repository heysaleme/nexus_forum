package storage

import (
	"context"
	"io"
	"testing"
)

type stubAccessor struct {
	called string
	out    string
}

func (s *stubAccessor) Backend() Backend { return BackendMinIO }

func (s *stubAccessor) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	return "", nil
}

func (s *stubAccessor) AccessibleURL(ctx context.Context, reference string) (string, error) {
	s.called = reference
	return s.out, nil
}

func TestResolveMediaJSON(t *testing.T) {
	store := &stubAccessor{out: "https://signed.example/obj"}
	raw := `["http://localhost:9000/nexus-forum/posts/videos/a.mp4"]`
	got := ResolveMediaJSON(context.Background(), store, raw)
	want := `["https://signed.example/obj"]`
	if got != want {
		t.Fatalf("got %q want %q (called %q)", got, want, store.called)
	}
}
