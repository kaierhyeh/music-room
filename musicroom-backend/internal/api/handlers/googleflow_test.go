package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestResolveReturn(t *testing.T) {
	h := NewGoogleFlowHandler(nil, nil, "musicroom://auth")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty falls back", "", "musicroom://auth"},
		{"app scheme", "musicroom://auth", "musicroom://auth"},
		{"expo go on the lan", "exp://192.168.1.2:8082/--/auth", "exp://192.168.1.2:8082/--/auth"},
		{"expo go on loopback", "exp://127.0.0.1:8082/--/auth", "exp://127.0.0.1:8082/--/auth"},
		{"expo go on localhost", "exp://localhost:8082/--/auth", "exp://localhost:8082/--/auth"},
		{"expo server on the internet", "exp://203.0.113.7:8082/--/auth", "musicroom://auth"},
		{"web on localhost", "http://localhost:19006/auth", "http://localhost:19006/auth"},
		{"web on loopback", "http://127.0.0.1:19006/auth", "http://127.0.0.1:19006/auth"},
		{"web on the lan", "http://192.168.1.2:19006/auth", "http://192.168.1.2:19006/auth"},
		{"web over tls on the lan", "https://192.168.1.2:19006/auth", "https://192.168.1.2:19006/auth"},
		{"web on a public host", "https://evil.example/steal", "musicroom://auth"},
		{"web on a public ip", "http://203.0.113.7/steal", "musicroom://auth"},
		{"foreign scheme", "ftp://192.168.1.2/steal", "musicroom://auth"},
		{"scheme-less", "//evil.example/steal", "musicroom://auth"},
		{"unparseable", "://", "musicroom://auth"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := h.resolveReturn(tc.in, false); got != tc.want {
				t.Fatalf("resolveReturn(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestReturnStoreIsSingleUse(t *testing.T) {
	s := newReturnStore(time.Minute)
	tok, err := s.Issue("exp://192.168.1.2:8082/--/auth", uuid.Nil, "")
	if err != nil {
		t.Fatal(err)
	}
	got, _, _, ok := s.Consume(tok)
	if !ok || got != "exp://192.168.1.2:8082/--/auth" {
		t.Fatalf("first consume = %q, %v", got, ok)
	}
	if _, _, _, ok := s.Consume(tok); ok {
		t.Fatal("state token was accepted twice; a callback could be replayed")
	}
	if _, _, _, ok := s.Consume(""); ok {
		t.Fatal("empty state accepted")
	}
	if _, _, _, ok := s.Consume("not-a-token"); ok {
		t.Fatal("unknown state accepted")
	}
}

func TestReturnStoreCarriesTheLinkTarget(t *testing.T) {
	s := newReturnStore(time.Minute)
	user := uuid.New()

	tok, err := s.Issue("musicroom://auth", user, "")
	if err != nil {
		t.Fatal(err)
	}
	_, linkTo, _, ok := s.Consume(tok)
	if !ok {
		t.Fatal("consume failed")
	}
	if linkTo != user {
		t.Fatalf("link target = %v, want %v", linkTo, user)
	}

	plain, err := s.Issue("musicroom://auth", uuid.Nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, linkTo, _, _ := s.Consume(plain); linkTo != uuid.Nil {
		t.Fatalf("a sign-in must carry no link target, got %v", linkTo)
	}
}

func TestReturnStoreExpires(t *testing.T) {
	s := newReturnStore(-time.Second)
	tok, err := s.Issue("musicroom://auth", uuid.Nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := s.Consume(tok); ok {
		t.Fatal("expired state accepted")
	}
}

func TestResolveReturnWithPKCE(t *testing.T) {
	h := NewGoogleFlowHandler(nil, nil, "musicroom://auth")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"expo go behind a tunnel", "exp://abc-def.trycloudflare.com/--/auth", "exp://abc-def.trycloudflare.com/--/auth"},
		{"expo go on the lan", "exp://192.168.1.2:8082/--/auth", "exp://192.168.1.2:8082/--/auth"},
		{"exp without a host", "exp:///--/auth", "musicroom://auth"},
		{"web on a public host", "https://evil.example/steal", "musicroom://auth"},
		{"web on the lan", "http://192.168.1.2:19006/auth", "http://192.168.1.2:19006/auth"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := h.resolveReturn(tc.in, true); got != tc.want {
				t.Fatalf("resolveReturn(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
	if got := h.resolveReturn("exp://abc-def.trycloudflare.com/--/auth", false); got != "musicroom://auth" {
		t.Fatalf("a tunnel host without PKCE must fall back, got %q", got)
	}
}

func s256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func TestVerifierMatches(t *testing.T) {
	verifier := strings.Repeat("ab", 32)
	challenge := s256(verifier)

	if !validChallenge(challenge) {
		t.Fatalf("a real S256 challenge was rejected: %q", challenge)
	}
	if !verifierMatches(verifier, challenge) {
		t.Fatal("the right verifier was rejected")
	}
	if verifierMatches(strings.Repeat("cd", 32), challenge) {
		t.Fatal("a wrong verifier was accepted")
	}
	if verifierMatches("", challenge) {
		t.Fatal("an empty verifier was accepted")
	}
	short := "tooshort"
	if verifierMatches(short, s256(short)) {
		t.Fatal("a verifier under the RFC 7636 minimum was accepted")
	}
	for _, bad := range []string{"", "plain-text", challenge + "=", verifier} {
		if validChallenge(bad) {
			t.Fatalf("validChallenge(%q) = true", bad)
		}
	}
}

func TestChallengeSurvivesTheRoundTrip(t *testing.T) {
	states := newReturnStore(time.Minute)
	tok, err := states.Issue("exp://abc.trycloudflare.com/--/auth", uuid.Nil, "the-challenge")
	if err != nil {
		t.Fatal(err)
	}
	_, _, challenge, ok := states.Consume(tok)
	if !ok || challenge != "the-challenge" {
		t.Fatalf("state carried challenge %q, %v", challenge, ok)
	}

	codes := newHandoffStore(time.Minute)
	user := uuid.New()
	code, err := codes.Issue(user, challenge)
	if err != nil {
		t.Fatal(err)
	}
	gotUser, gotChallenge, ok := codes.Consume(code)
	if !ok || gotUser != user || gotChallenge != "the-challenge" {
		t.Fatalf("code consume = %v, %q, %v", gotUser, gotChallenge, ok)
	}
	if _, _, ok := codes.Consume(code); ok {
		t.Fatal("one-time code was accepted twice")
	}
}
