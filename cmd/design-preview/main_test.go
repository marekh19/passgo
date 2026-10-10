package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreviewRoutes(t *testing.T) {
	handler := previewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "js/payment.js" {
			t.Errorf("static path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/", 200}, {"GET", "/static/js/payment.js", 200},
		{"POST", "/", 405}, {"POST", "/static/js/payment.js", 405},
		{"GET", "/games", 404}, {"POST", "/pay", 404},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("status %d, want %d", w.Code, tc.status)
			}
		})
	}
}

func TestPreviewMarkup(t *testing.T) {
	w := httptest.NewRecorder()
	previewHandler(http.NotFoundHandler()).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	for _, fragment := range []string{
		`lang="en"`, `data-sheet-open="demo-payment"`, `aria-labelledby="demo-payment-title"`,
		`for="demo-amount"`, `aria-describedby="demo-amount-hint demo-amount-error"`,
		`aria-label="Backspace"`, `aria-label="Close payment sheet"`, `inputmode="none"`,
		`aria-pressed="false"`, `role="status"`, `$18,446,744,073,709,551,615`, `ABCD`,
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("missing %s", fragment)
		}
	}
	for _, forbidden := range []string{`method="post"`, `action="/`, `live.js`, `htmx`, `data-live`} {
		if strings.Contains(body, forbidden) {
			t.Errorf("unexpected %s", forbidden)
		}
	}
}
