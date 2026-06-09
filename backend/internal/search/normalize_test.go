package search

import "testing"

func TestStemTokenRussianVariants(t *testing.T) {
	if StemToken("ленты") != StemToken("лента") {
		t.Fatalf("expected ленты/лента to share stem, got %q vs %q", StemToken("ленты"), StemToken("лента"))
	}
}

func TestNormalizeFoldCyrillicCase(t *testing.T) {
	if NormalizeFold("ЛЕНТА") != NormalizeFold("лента") {
		t.Fatal("case fold failed for Cyrillic")
	}
}

func TestTokenizeMixedText(t *testing.T) {
	tokens := Tokenize("Hello дизайн UI-v2")
	if len(tokens) < 3 {
		t.Fatalf("expected mixed tokens, got %v", tokens)
	}
}
