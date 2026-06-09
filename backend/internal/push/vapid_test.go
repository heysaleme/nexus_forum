package push

import "testing"

func TestNormalizeSubscriber_StripsMailtoPrefix(t *testing.T) {
	got := NormalizeSubscriber("mailto:user@example.com")
	if got != "user@example.com" {
		t.Fatalf("got %q want user@example.com", got)
	}
	if JWTSubjectClaim(got) != "mailto:user@example.com" {
		t.Fatalf("jwt sub wrong: %s", JWTSubjectClaim(got))
	}
}

func TestNormalizeSubscriber_FixesDoubleMailto(t *testing.T) {
	got := NormalizeSubscriber("mailto:mailto:user@example.com")
	if got != "user@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateConfig_KeyPair(t *testing.T) {
	priv := "CVz4N2_gDbqAfCYchvdOrkWvCs66pKjAo05UXkMRoVE"
	pub := "BATaPFp5XCA0UcVLLiaCY-yq9AysQvKDcQKsnEIq7LBat0BqV-83l8-9ZnKisIbkeGAyf_JUp3DMJmPagsJred8"
	cfg, err := ValidateConfig(pub, priv, "mailto:mikakkumi@gmail.com")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !cfg.KeysMatch {
		t.Fatal("keys should match")
	}
	if cfg.Subscriber != "mikakkumi@gmail.com" {
		t.Fatalf("subscriber %q", cfg.Subscriber)
	}
	if cfg.JWTSubject != "mailto:mikakkumi@gmail.com" {
		t.Fatalf("jwt subject %q", cfg.JWTSubject)
	}
}

func TestValidateConfig_MismatchFails(t *testing.T) {
	_, err := ValidateConfig("wrong-public-key", "CVz4N2_gDbqAfCYchvdOrkWvCs66pKjAo05UXkMRoVE", "a@b.com")
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}
