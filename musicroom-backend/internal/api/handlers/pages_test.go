package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestPublicPages(t *testing.T) {
	e := echo.New()
	for _, tc := range []struct {
		name string
		h    echo.HandlerFunc
		want string
	}{
		{"home", HomePage, "MusicRoom"},
		{"privacy", PrivacyPage, "Privacy policy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
			if err := tc.h(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("got %d, body without %q", rec.Code, tc.want)
			}
			if ct := rec.Header().Get(echo.HeaderContentType); !strings.HasPrefix(ct, "text/html") {
				t.Fatalf("content type %q", ct)
			}
		})
	}
}
