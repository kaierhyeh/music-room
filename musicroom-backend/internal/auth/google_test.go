package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyRefusesWithoutAClientID(t *testing.T) {
	reached := false
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))
	defer upstream.Close()

	v := NewGoogleVerifier("")
	claims, err := v.Verify(context.Background(), "any.id.token")
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
	if claims != nil {
		t.Fatal("no claims may be returned when the audience cannot be checked")
	}
	if reached {
		t.Fatal("the token was sent upstream before the guard ran")
	}
}
