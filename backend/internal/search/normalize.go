package search

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var lower = cases.Lower(language.Und, cases.NoLower)

// NormalizeFold lowercases text with Unicode case folding (works for Cyrillic and Latin).
func NormalizeFold(s string) string {
	return lower.String(strings.TrimSpace(s))
}

// Tokenize splits text into searchable words (letters and digits, any script).
func Tokenize(s string) []string {
	s = NormalizeFold(s)
	var tokens []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			tokens = append(tokens, b.String())
			b.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return tokens
}

// Russian/English light stemmer for morphology-aware prefix matching.
func StemToken(token string) string {
	token = NormalizeFold(token)
	if token == "" {
		return ""
	}
	suffixes := []string{
		"иями", "ями", "ами", "ого", "ему", "ому", "ыми", "ими", "ией",
		"ий", "ый", "ой", "ая", "ое", "ые", "ие", "ов", "ев", "ам", "ям",
		"ах", "ях", "ию", "ью", "ия", "es", "ed", "ing", "ion", "ment",
		"а", "я", "ы", "и", "е", "о", "у", "ю", "ь", "й", "s",
	}
	for _, suf := range suffixes {
		if len(token) > len(suf)+1 && strings.HasSuffix(token, suf) {
			return token[:len(token)-len(suf)]
		}
	}
	return token
}

// BuildBlob returns a normalized searchable blob for substring matching.
func BuildBlob(parts ...string) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(NormalizeFold(p))
	}
	return b.String()
}

// IndexTerms returns unique raw tokens and stems for a document.
func IndexTerms(parts ...string) (tokens []string, stems []string) {
	seenTok := map[string]struct{}{}
	seenStem := map[string]struct{}{}
	for _, part := range parts {
		for _, tok := range Tokenize(part) {
			if len(tok) < 2 {
				continue
			}
			if _, ok := seenTok[tok]; !ok {
				seenTok[tok] = struct{}{}
				tokens = append(tokens, tok)
			}
			stem := StemToken(tok)
			if len(stem) < 2 {
				continue
			}
			if _, ok := seenStem[stem]; !ok {
				seenStem[stem] = struct{}{}
				stems = append(stems, stem)
			}
		}
	}
	return tokens, stems
}
