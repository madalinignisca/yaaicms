// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverer(t *testing.T) {
	t.Run("catches panic and returns 500", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something went wrong")
		})

		handler := Recoverer(inner)

		req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
		rr := httptest.NewRecorder()

		// Should NOT panic — the middleware catches it.
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("status: got %d, want 500", rr.Code)
		}

		body := rr.Body.String()
		if !strings.Contains(body, "Internal Server Error") {
			t.Errorf("body: got %q, want it to contain %q", body, "Internal Server Error")
		}
	})

	t.Run("catches panic with integer value", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(42)
		})

		handler := Recoverer(inner)

		req := httptest.NewRequest(http.MethodGet, "/crash", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("status: got %d, want 500", rr.Code)
		}
	})

	t.Run("catches panic with error value", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(strings.NewReader("error reader")) // arbitrary non-string panic
		})

		handler := Recoverer(inner)

		req := httptest.NewRequest(http.MethodGet, "/crash", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("status: got %d, want 500", rr.Code)
		}
	})
}

func TestRecovererNoPanic(t *testing.T) {
	t.Run("normal pass-through without panic", func(t *testing.T) {
		var called bool
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		handler := Recoverer(inner)

		req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !called {
			t.Error("next handler should have been called")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("status: got %d, want 200", rr.Code)
		}
		if rr.Body.String() != "ok" {
			t.Errorf("body: got %q, want %q", rr.Body.String(), "ok")
		}
	})

	t.Run("preserves response headers", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "test-value")
			w.WriteHeader(http.StatusOK)
		})

		handler := Recoverer(inner)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if got := rr.Header().Get("X-Custom"); got != "test-value" {
			t.Errorf("X-Custom: got %q, want %q", got, "test-value")
		}
	})

	t.Run("works with different HTTP methods", func(t *testing.T) {
		methods := []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
		}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				handler := Recoverer(inner)

				req := httptest.NewRequest(method, "/admin/content", nil)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				if rr.Code != http.StatusOK {
					t.Errorf("status: got %d, want 200", rr.Code)
				}
			})
		}
	})
}
