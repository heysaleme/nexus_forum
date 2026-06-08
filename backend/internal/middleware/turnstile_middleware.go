package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// turnstileVerifyURL is Cloudflare's server-side siteverify endpoint.
const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type turnstileResponse struct {
	Success bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// VerifyTurnstileToken validates a Cloudflare Turnstile token server-side.
//
// Returns (true, nil) if the token is valid.
// Returns (true, nil) also when secret is empty — this means Turnstile is disabled/unconfigured.
// Returns (false, err) if the token is invalid or the verification request fails.
func VerifyTurnstileToken(secret, token, remoteIP string) (bool, error) {
	if secret == "" {
		// Turnstile not configured — silently pass all requests
		return true, nil
	}

	form := url.Values{
		"secret":   {secret},
		"response": {token},
	}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	resp, err := http.PostForm(turnstileVerifyURL, form)
	if err != nil {
		return false, fmt.Errorf("turnstile: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("turnstile: read response: %w", err)
	}

	var result turnstileResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return false, fmt.Errorf("turnstile: parse response: %w", err)
	}

	return result.Success, nil
}
