package storage

import (
	"context"
	"testing"
)

func TestMinIOStore_ObjectKeyFromReference(t *testing.T) {
	s := &MinIOStore{
		bucket:    "nexus-forum",
		publicURL: "http://localhost:9000",
	}
	ref := "http://localhost:9000/nexus-forum/posts/videos/123_test.mp4"
	key, ok := s.ObjectKeyFromReference(ref)
	if !ok || key != "posts/videos/123_test.mp4" {
		t.Fatalf("got key=%q ok=%v", key, ok)
	}
}

func TestReferenceURL(t *testing.T) {
	got := ReferenceURL("http://localhost:9000", "nexus-forum", "posts/images/a.gif")
	want := "http://localhost:9000/nexus-forum/posts/images/a.gif"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPublicEndpointFromURL(t *testing.T) {
	host, secure, ok := publicEndpointFromURL("http://localhost:9000")
	if !ok || host != "localhost:9000" || secure {
		t.Fatalf("got host=%q secure=%v ok=%v", host, secure, ok)
	}
	host, secure, ok = publicEndpointFromURL("https://cdn.example.com")
	if !ok || host != "cdn.example.com" || !secure {
		t.Fatalf("got host=%q secure=%v ok=%v", host, secure, ok)
	}
}

func TestObjectKeyFromReference_PresignedURL(t *testing.T) {
	s := &MinIOStore{
		bucket:    "nexus-forum",
		publicURL: "http://localhost:9000",
	}
	ref := "http://localhost:9000/nexus-forum/posts/videos/clip.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Signature=abc"
	key, ok := s.ObjectKeyFromReference(ref)
	if !ok || key != "posts/videos/clip.mp4" {
		t.Fatalf("got key=%q ok=%v", key, ok)
	}
}

func TestCanonicalMediaReference(t *testing.T) {
	raw := "http://localhost:9000/nexus-forum/posts/videos/a.mp4?X-Amz-Signature=abc"
	got := CanonicalMediaReference(raw)
	want := "http://localhost:9000/nexus-forum/posts/videos/a.mp4"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestLocalStore_AccessibleURL(t *testing.T) {
	s := &LocalStore{publicURL: "http://localhost:8080"}
	got, err := s.AccessibleURL(context.Background(), "/uploads/posts/images/x.png")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://localhost:8080/uploads/posts/images/x.png" {
		t.Fatalf("got %q", got)
	}
}
