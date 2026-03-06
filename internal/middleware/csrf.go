// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

const (
	// csrfTokenLength is the byte length of CSRF tokens (32 bytes = 64 hex chars).
	csrfTokenLength = 32

	// CSRFCookieName is the cookie that holds the CSRF token.
	CSRFCookieName = "sp_csrf"

	// CSRFHeaderName is the header HTMX sends the CSRF token in.
	// Configured via hx-headers in the admin layout.
	CSRFHeaderName = "X-CSRF-Token"

	// CSRFFormField is the hidden form field name for non-HTMX forms.
	CSRFFormField = "csrf_token"

	// csrfContextKey stores the CSRF token in the request context so
	// templates can access it even on the first request (before the
	// cookie round-trips back from the browser).
	csrfContextKey contextKey = "csrf_token"
)

// NewCSRF returns a CSRF middleware with the cookie Secure flag controlled
// by the secure parameter. Set secure=true in production (behind TLS).
func NewCSRF(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			// Ensure a CSRF token cookie exists.
			cookie, err := r.Cookie(CSRFCookieName)
			if err != nil || cookie.Value == "" {
				token, err = generateCSRFToken()
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:     CSRFCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // JS needs to read this for HTMX hx-headers
					Secure:   secure,
					SameSite: http.SameSiteStrictMode,
				})
			} else {
				token = cookie.Value
			}

			// Store the token in context so templates can always access it.
			ctx := context.WithValue(r.Context(), csrfContextKey, token)
			r = r.WithContext(ctx)

			// Safe methods don't need CSRF validation.
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// For state-changing methods, validate the token.
			// Check header first (HTMX), then form field.
			submitted := r.Header.Get(CSRFHeaderName)
			if submitted == "" {
				submitted = r.FormValue(CSRFFormField)
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(submitted)) != 1 {
				http.Error(w, "CSRF token mismatch", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CSRFTokenFromCtx extracts the CSRF token from the request context.
// This is the preferred way to get the token in templates — it works
// even on the first request before the cookie round-trips.
func CSRFTokenFromCtx(ctx context.Context) string {
	token, _ := ctx.Value(csrfContextKey).(string)
	return token
}

// generateCSRFToken creates a cryptographically random token.
func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
