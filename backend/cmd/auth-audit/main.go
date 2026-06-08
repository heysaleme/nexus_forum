package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type loginResp struct {
	AccessToken string `json:"access_token"`
}

func main() {
	base := envOr("API_BASE", "http://localhost:8080/api")

	users := []struct {
		label string
		email string
	}{
		{"user", envOr("AUDIT_USER_EMAIL", "kai@example.com")},
		{"moderator", envOr("AUDIT_MOD_EMAIL", "moderator@example.com")},
		{"admin", envOr("AUDIT_ADMIN_EMAIL", "amira@example.com")},
	}

	tokens := map[string]string{}
	for _, u := range users {
		tok, err := login(base, u.email, "password123")
		if err != nil {
			fmt.Printf("LOGIN %-12s FAIL %v\n", u.label, err)
			continue
		}
		tokens[u.label] = tok
		fmt.Printf("LOGIN %-12s OK\n", u.label)
	}

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/moderation/reports", ""},
		{"PUT", "/moderation/reports/1", `{"status":"dismissed","moderator_response":"audit"}`},
		{"POST", "/moderation/users/2/shadow-ban", `{"reason":"audit"}`},
		{"POST", "/moderation/users/2/unshadow-ban", `{"reason":"audit"}`},
		{"POST", "/moderation/filters", `{"pattern":"test","action":"block","is_regex":false}`},
		{"DELETE", "/moderation/filters/1", ""},
		{"GET", "/moderation/communities/1/logs", ""},
		{"GET", "/moderation/logs", ""},
		{"GET", "/analytics/dashboard", ""},
	}

	fmt.Println("\n=== RBAC AUDIT ===")
	for _, ep := range endpoints {
		fmt.Printf("\n%s %s\n", ep.method, ep.path)
		for _, role := range []string{"user", "moderator", "admin"} {
			tok := tokens[role]
			if tok == "" {
				fmt.Printf("  %-10s SKIP (no token)\n", role)
				continue
			}
			code, snippet := request(base, ep.method, ep.path, tok, ep.body)
			fmt.Printf("  %-10s HTTP %d %s\n", role, code, snippet)
		}
	}
}

func login(base, email, password string) (string, error) {
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	resp, err := http.Post(base+"/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(raw))
	}
	var lr loginResp
	if err := json.Unmarshal(raw, &lr); err != nil {
		return "", err
	}
	return lr.AccessToken, nil
}

func request(base, method, path, token, body string) (int, string) {
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req, err := http.NewRequest(method, base+path, reader)
	if err != nil {
		return 0, err.Error()
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	snippet := strings.TrimSpace(string(raw))
	if len(snippet) > 80 {
		snippet = snippet[:80] + "..."
	}
	return resp.StatusCode, snippet
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
