package service

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
	pushcfg "nexus-forum-backend/internal/push"
)

func jwtPayloadSub(authHeader string) (string, error) {
	// Authorization: vapid t=<jwt>, k=<publicKey>
	const prefix = "vapid t="
	idx := strings.Index(authHeader, prefix)
	if idx < 0 {
		return "", nil
	}
	rest := authHeader[idx+len(prefix):]
	comma := strings.Index(rest, ",")
	if comma < 0 {
		return "", nil
	}
	token := rest[:comma]
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var claims struct {
		Sub string `json:"sub"`
		Aud string `json:"aud"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return "", err
	}
	return claims.Sub, nil
}

func TestSendPayload_VAPIDJWTSubject_NotDoubleMailto(t *testing.T) {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate keys: %v", err)
	}
	vapid, err := pushcfg.ValidateConfig(pub, priv, "mailto:mikakkumi@gmail.com")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	resp, sendErr := webpush.SendNotification([]byte(`{"title":"t"}`), &webpush.Subscription{
		Endpoint: srv.URL,
		Keys:     webpush.Keys{P256dh: pub, Auth: "auth-secret-16b"},
	}, &webpush.Options{
		Subscriber:      vapid.Subscriber,
		VAPIDPublicKey:  vapid.PublicKey,
		VAPIDPrivateKey: vapid.PrivateKey,
		TTL:             60,
	})
	if sendErr != nil {
		t.Fatalf("send: %v", sendErr)
	}
	_ = resp.Body.Close()

	subClaim, err := jwtPayloadSub(authHeader)
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}
	if subClaim != "mailto:mikakkumi@gmail.com" {
		t.Fatalf("jwt sub=%q want mailto:mikakkumi@gmail.com (double mailto: causes BadJwtToken)", subClaim)
	}

	// Old bug: passing mailto: prefix through to webpush-go produced mailto:mailto:...
	badSub := "mailto:mikakkumi@gmail.com"
	if !strings.HasPrefix(badSub, "https:") {
		badSub = "mailto:" + badSub
	}
	if badSub == "mailto:mikakkumi@gmail.com" {
		t.Fatal("sanity check failed")
	}
	if badSub != "mailto:mailto:mikakkumi@gmail.com" {
		t.Fatalf("expected double mailto bug demonstration, got %q", badSub)
	}
}

func TestSendPayload_MockAppleAndFCM_Return201(t *testing.T) {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate keys: %v", err)
	}
	vapid, err := pushcfg.ValidateConfig(pub, priv, "mikakkumi@gmail.com")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	cases := []struct {
		name     string
		provider string
		path     string
	}{
		{"apple", "apple", "/apple-push"},
		{"fcm", "fcm", "/fcm-send"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path {
					http.NotFound(w, r)
					return
				}
				sub, _ := jwtPayloadSub(r.Header.Get("Authorization"))
				if sub != vapid.JWTSubject {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"reason":"BadJwtToken"}`))
					return
				}
				w.WriteHeader(http.StatusCreated)
			}))
			defer srv.Close()

			endpoint := srv.URL + tc.path
			resp, err := webpush.SendNotification([]byte(`{"title":"Nexus Forum","body":"test"}`), &webpush.Subscription{
				Endpoint: endpoint,
				Keys:     webpush.Keys{P256dh: pub, Auth: "auth-secret-16b"},
			}, &webpush.Options{
				Subscriber:      vapid.Subscriber,
				VAPIDPublicKey:  vapid.PublicKey,
				VAPIDPrivateKey: vapid.PrivateKey,
				TTL:             60,
			})
			if err != nil {
				t.Fatalf("send: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("http_status=%d want 201", resp.StatusCode)
			}
		})
	}
}
