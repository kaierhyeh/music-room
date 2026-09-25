package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type GoogleClaims struct {
	Sub           string
	Email         string
	EmailVerified bool
	Name          string
}

var ErrNotConfigured = errors.New("google sign-in is not configured")

type GoogleVerifier interface {
	Verify(ctx context.Context, idToken string) (*GoogleClaims, error)
}

type tokenInfoVerifier struct {
	clientID string
	client   *http.Client
}

func NewGoogleVerifier(clientID string) GoogleVerifier {
	return &tokenInfoVerifier{clientID: clientID, client: &http.Client{Timeout: 5 * time.Second}}
}

func (v *tokenInfoVerifier) Verify(ctx context.Context, idToken string) (*GoogleClaims, error) {
	if v.clientID == "" {
		return nil, ErrNotConfigured
	}
	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google: invalid id_token (status %d)", resp.StatusCode)
	}

	var payload struct {
		Aud           string `json:"aud"`
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Aud != v.clientID {
		return nil, fmt.Errorf("google: unexpected audience")
	}
	if payload.Sub == "" {
		return nil, fmt.Errorf("google: missing sub claim")
	}
	return &GoogleClaims{
		Sub:           payload.Sub,
		Email:         payload.Email,
		EmailVerified: payload.EmailVerified == "true",
		Name:          payload.Name,
	}, nil
}
