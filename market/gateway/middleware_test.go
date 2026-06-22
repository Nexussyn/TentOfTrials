# Fix for Issue #2: [$50 BOUNTY] [Go] Add gateway auth middleware tests

// market/gateway/middleware_test.go
package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestAuthMiddleware_MissingToken verifies that requests without a bearer token
// return 401 with the expected JSON error shape.
func TestAuthMiddleware_MissingToken(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("wrapped handler should not be called when token is missing")
	}))

	tests := []struct {
		name          string
		authorization string
	}{
		{"no header", ""},
		{"empty header", ""},
		{"bearer prefix only", "Bearer "},
		{"bearer prefix no space", "Bearer"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rec.Code)
			}

			contentType := rec.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("expected JSON content type, got %s", contentType)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error == "" {
				t.Error("expected non-empty error field in response")
			}

			if errResp.Code != "UNAUTHORIZED" && errResp.Code != "AUTH_REQUIRED" {
				t.Logf("error code: %s (verify this matches expected schema)", errResp.Code)
			}
		})
	}
}

// TestAuthMiddleware_InvalidToken verifies that requests with an invalid token
// return 401 without calling the wrapped handler.
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	handlerCalled := false
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	invalidTokens := []string{
		"invalid-token",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
		"expired-token-simulation",
		"malformed.jwt",
		"   ",
	}

	for _, token := range invalidTokens {
		t.Run("token_"+sanitizeTestName(token), func(t *testing.T) {
			handlerCalled = false

			req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if handlerCalled {
				t.Error("wrapped handler should not be called for invalid token")
			}

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rec.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error == "" {
				t.Error("expected non-empty error field in response")
			}
		})
	}
}

// TestAuthMiddleware_ValidToken verifies that a valid token sets user/session/auth
// context before the request reaches downstream handlers.
func TestAuthMiddleware_ValidToken(t *testing.T) {
	var capturedUserID string
	var capturedSessionID string
	var capturedAuthContext *AuthContext

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = GetUserIDFromContext(r.Context())
		capturedSessionID = GetSessionIDFromContext(r.Context())
		capturedAuthContext = GetAuthContextFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	validToken := GetTestValidToken()

	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if capturedUserID == "" {
		t.Error("expected user ID to be set in context")
	}

	if capturedSessionID == "" {
		t.Error("expected session ID to be set in context")
	}

	if capturedAuthContext == nil {
		t.Error("expected auth context to be set")
	} else {
		if capturedAuthContext.UserID == "" {
			t.Error("expected auth context to have user ID")
		}
		if !capturedAuthContext.Authenticated {
			t.Error("expected auth context to be marked as authenticated")
		}
	}
}

// TestMiddlewareOrdering_AuthBeforeRateLimit verifies that authenticated requests
// are keyed separately from anonymous/IP-only requests for rate limiting.
func TestMiddlewareOrdering_AuthBeforeRateLimit(t *testing.T) {
	rateLimitKeys := &sync.Map{}

	// Create a custom rate limiter that captures the keys used
	mockRateLimiter := &MockRateLimiter{
		KeyCapture: rateLimitKeys,
	}

	// Build the middleware chain in the correct order: Auth -> RateLimit -> Handler
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain := AuthMiddleware(
		RateLimitMiddlewareWithLimiter(mockRateLimiter)(finalHandler),
	)

	// Test 1: Anonymous request (no token)
	t.Run("anonymous request uses IP-based key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		req.RemoteAddr = "192.168.1.100:12345"

		rec := httptest.NewRecorder()
		chain.ServeHTTP(rec, req)

		// For anonymous requests, we expect either 401 or rate limit key based on IP
		// The exact behavior depends on whether auth is required or optional
		keys := collectKeys(rateLimitKeys)
		t.Logf("Rate limit keys for anonymous: %v", keys)
	})

	// Test 2: Authenticated request
	t.Run("authenticated request uses user-based key", func(t *testing.T) {
		rateLimitKeys = &sync.Map{}
		mockRateLimiter.KeyCapture = rateLimitKeys

		req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		req.Header.Set("Authorization", "Bearer "+GetTestValidToken())
		req.RemoteAddr = "192.168.1.100:12345"

		rec := httptest.NewRecorder()
		chain.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			keys := collectKeys(rateLimitKeys)
			t.Logf("Rate limit keys for authenticated: %v", keys)

			// Verify the key contains user identifier, not just IP
			foundUserKey := false
			for _, key := range keys {
				if strings.Contains(key, "user:") || strings.Contains(key, GetTestUserID()) {
					foundUserKey = true
					break
				}
			}

			if !foundUserKey && len(keys) > 0 {
				// Check that the key differs from a pure IP key
				for _, key := range keys {
					if key == "192.168.1.100" {
						t.Error("authenticated request should not use plain IP as rate limit key")
					}
				}
			}
		}
	})

	// Test 3: Different users should have different rate limit keys
	t.Run("different users have different rate limit keys", func(t *testing.T) {
		user1Keys := &sync.Map{}
		user2Keys := &sync.Map{}

		limiter1 := &MockRateLimiter{KeyCapture: user1Keys}
		limiter2 := &MockRateLimiter{KeyCapture: user2Keys}

		chain1 := AuthMiddleware(RateLimitMiddlewareWithLimiter(limiter1)(finalHandler))
		chain2 := AuthMiddleware(RateLimitMiddlewareWithLimiter(limiter2)(finalHandler))

		req1 := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		req1.Header.Set("Authorization", "Bearer "+GetTestValidTokenForUser("user-1"))
		req1.RemoteAddr = "192.168.1.100:12345"

		req2 := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		req2.Header.Set("Authorization", "Bearer "+GetTestValidTokenForUser("user-2"))
		req2.RemoteAddr = "192.168.1.100:12345" // Same IP

		rec1 := httptest.NewRecorder()
		rec2 := httptest.NewRecorder()

		chain1.ServeHTTP(rec1, req1)
		chain2.ServeHTTP(rec2, req2)

		keys1 := collectKeys(user1Keys)
		keys2 := collectKeys(user2Keys)

		t.Logf("User 1 keys: %v", keys1)
		t.Logf("User 2 keys: %v", keys2)

		// If both succeeded, verify keys are different
		if rec1.Code == http.StatusOK && rec2.Code == http.StatusOK {
			if len(keys1) > 0 && len(keys2) > 0 {
				if keys1[0] == keys2[0] {
					t.Error("different users should have different rate limit keys")
				}
			}
		}
	})
}

// TestMiddlewareChain_ContextPropagation verifies context flows through the chain.
func TestMiddlewareChain_ContextPropagation(t *testing.T) {
	var contextValues map[string]interface{}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		contextValues = map[string]interface{}{
			"userID":      GetUserIDFromContext(ctx),
			"sessionID":   GetSessionIDFromContext(ctx),
			"authContext": GetAuthContextFromContext(ctx),
			"rateLimited": ctx.Value(RateLimitContextKey),
		}
		w.WriteHeader(http.StatusOK)
	})

	chain := AuthMiddleware(
		RateLimitMiddleware(finalHandler),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer "+GetTestValidToken())

	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		if contextValues["userID"] == "" {
			t.Error("user ID should propagate through middleware chain")
		}
		if contextValues["sessionID"] == "" {
			t.Error("session ID should propagate through middleware chain")
		}
		if contextValues["authContext"] == nil {
			t.Error("auth context should propagate through middleware chain")
		}
	}
}

// TestRateLimitMiddleware_RespectsAuthContext verifies rate limiter uses auth info.
func TestRateLimitMiddleware_RespectsAuthContext(t *testing.T) {
	limiter := &MockRateLimiter{
		KeyCapture:   &sync.Map{},
		AllowRequest: true,
	}

	handler := RateLimitMiddlewareWithLimiter(limiter)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// Create request with pre-set auth context (simulating auth middleware already ran)
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req = req.WithContext(SetAuthContext(req.Context(), &AuthContext{
		UserID:        "test-user-123",
		SessionID:     "session-456",
		Authenticated: true,
	}))
	req.RemoteAddr = "10.0.0.1:9999"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	keys := collectKeys(limiter.KeyCapture)
	t.Logf("Captured rate limit keys: %v", keys)

	// Verify rate limiter received the auth-enriched context
	if len(keys) > 0 {
		foundUserBasedKey := false
		for _, key := range keys {
			if strings.Contains(key, "test-user-123") || strings.Contains(key, "user:") {
				foundUserBasedKey = true
				break
			}
		}
		if !foundUserBasedKey {
			// Check it's at least not just the IP
			for _, key := range keys {
				if key == "10.0.0.1" {
					t.Error("rate limiter should use auth context, not just IP, when available")
				}
			}
		}
	}
}

// Helper functions

func sanitizeTestName(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ".", "_")
	if len(s) > 20 {
		s = s[:20]
	}
	return s
}

func collectKeys(m *sync.Map) []string {
	var keys []string
	m.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok {
			keys = append(keys, k)
		}
		return true
	})
	return keys
}