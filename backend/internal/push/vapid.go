package push

import (
	"crypto/elliptic"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config holds validated VAPID settings for webpush-go.
type Config struct {
	PublicKey         string
	PrivateKey        string
	Subscriber        string // value passed to webpush.Options.Subscriber (email or https URL, NOT mailto:)
	JWTSubject        string // actual JWT "sub" claim after webpush normalization
	ConfiguredPublic  string
	DerivedPublicKey  string
	KeysMatch         bool
}

// NormalizeSubscriber returns the value for webpush.Options.Subscriber.
// webpush-go prepends "mailto:" unless the value starts with "https:",
// so we must NOT pass an already-prefixed mailto: address.
func NormalizeSubscriber(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "admin@nexus-forum.local"
	}
	for strings.HasPrefix(s, "mailto:mailto:") {
		s = strings.TrimPrefix(s, "mailto:")
	}
	if strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") {
		if strings.HasPrefix(s, "http://") {
			s = "https://" + strings.TrimPrefix(s, "http://")
		}
		return s
	}
	return strings.TrimPrefix(s, "mailto:")
}

// JWTSubjectClaim returns the JWT "sub" claim webpush-go will emit.
func JWTSubjectClaim(subscriber string) string {
	if strings.HasPrefix(subscriber, "https:") {
		return subscriber
	}
	return "mailto:" + subscriber
}

func decodeVapidKey(key string) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("empty vapid key")
	}
	if bytes, err := base64.URLEncoding.DecodeString(key); err == nil {
		return bytes, nil
	}
	return base64.RawURLEncoding.DecodeString(key)
}

// DerivePublicKey derives the VAPID public key from a private key (base64url).
func DerivePublicKey(privateKeyB64 string) (string, error) {
	decoded, err := decodeVapidKey(privateKeyB64)
	if err != nil {
		return "", fmt.Errorf("decode private key: %w", err)
	}
	if len(decoded) != 32 {
		return "", fmt.Errorf("invalid private key length %d (expected 32)", len(decoded))
	}
	curve := elliptic.P256()
	px, py := curve.ScalarMult(curve.Params().Gx, curve.Params().Gy, decoded)
	public := elliptic.Marshal(curve, px, py)
	return base64.RawURLEncoding.EncodeToString(public), nil
}

// ValidateConfig validates the VAPID key pair and normalizes the subscriber.
func ValidateConfig(publicKey, privateKey, subject string) (*Config, error) {
	publicKey = strings.TrimSpace(publicKey)
	privateKey = strings.TrimSpace(privateKey)
	if publicKey == "" || privateKey == "" {
		return nil, errors.New("VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY are required")
	}

	derived, err := DerivePublicKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("derive public key: %w", err)
	}

	subscriber := NormalizeSubscriber(subject)
	cfg := &Config{
		PublicKey:        publicKey,
		PrivateKey:       privateKey,
		Subscriber:       subscriber,
		JWTSubject:       JWTSubjectClaim(subscriber),
		ConfiguredPublic: publicKey,
		DerivedPublicKey: derived,
		KeysMatch:        publicKey == derived,
	}
	if !cfg.KeysMatch {
		return cfg, fmt.Errorf("VAPID public/private key mismatch: configured=%s derived=%s", publicKey, derived)
	}
	return cfg, nil
}

// JWTAudience returns the JWT "aud" claim for a push endpoint.
func JWTAudience(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	return u.Scheme + "://" + u.Host, nil
}

// JWTExpiration returns the default VAPID JWT expiration (12h from now).
func JWTExpiration() time.Time {
	return time.Now().Add(12 * time.Hour)
}
