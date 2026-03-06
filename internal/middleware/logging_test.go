// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger(t *testing.T) {
	t.Run("calls next handler and returns correct status", func(t *testing.T) {
		var called bool
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		handler := Logger(inner)

		req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !called {
			t.Error("next handler should have been called")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("status: got %d, want 200", rr.Code)
		}
	})

	t.Run("captures non-200 status code", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		handler := Logger(inner)

		req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("status: got %d, want 404", rr.Code)
		}
	})

	t.Run("handles write without explicit WriteHeader", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Write body without calling WriteHeader — Go defaults to 200.
			_, _ = w.Write([]byte("hello"))
		})

		handler := Logger(inner)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status: got %d, want 200", rr.Code)
		}
		if rr.Body.String() != "hello" {
			t.Errorf("body: got %q, want %q", rr.Body.String(), "hello")
		}
	})

	t.Run("works with POST requests", func(t *testing.T) {
		var gotMethod string
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			w.WriteHeader(http.StatusCreated)
		})

		handler := Logger(inner)

		req := httptest.NewRequest(http.MethodPost, "/admin/content", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if gotMethod != http.MethodPost {
			t.Errorf("method: got %q, want %q", gotMethod, http.MethodPost)
		}
		if rr.Code != http.StatusCreated {
			t.Errorf("status: got %d, want 201", rr.Code)
		}
	})
}

// TestResponseWriter tests the responseWriter wrapper used by the Logger
// middleware to verify it correctly captures status codes.
func TestResponseWriter(t *testing.T) {
	t.Run("WriteHeader captures status code", func(t *testing.T) {
		rr := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusNotFound)

		if rw.statusCode != http.StatusNotFound {
			t.Errorf("statusCode: got %d, want 404", rw.statusCode)
		}
		if !rw.written {
			t.Error("written should be true after WriteHeader")
		}
	})

	t.Run("WriteHeader only captures first call", func(t *testing.T) {
		rr := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusNotFound)
		rw.WriteHeader(http.StatusInternalServerError) // Should be ignored.

		if rw.statusCode != http.StatusNotFound {
			t.Errorf("statusCode: got %d, want 404 (first call)", rw.statusCode)
		}
	})

	t.Run("Write sets default 200 status", func(t *testing.T) {
		rr := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

		n, err := rw.Write([]byte("test"))
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		if n != 4 {
			t.Errorf("bytes written: got %d, want 4", n)
		}
		if rw.statusCode != http.StatusOK {
			t.Errorf("statusCode: got %d, want 200", rw.statusCode)
		}
		if !rw.written {
			t.Error("written should be true after Write")
		}
	})

	t.Run("Write does not override explicit WriteHeader", func(t *testing.T) {
		rr := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusCreated)
		_, _ = rw.Write([]byte("created"))

		if rw.statusCode != http.StatusCreated {
			t.Errorf("statusCode: got %d, want 201", rw.statusCode)
		}
	})
}
