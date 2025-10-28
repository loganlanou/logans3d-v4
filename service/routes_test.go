package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTier1_CriticalPublicRoutes tests that critical public routes exist and are accessible
func TestTier1_CriticalPublicRoutes(t *testing.T) {
	e, _ := setupTestEcho(t)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		// Core pages
		{"Home page", "GET", "/", http.StatusOK},
		{"Health check", "GET", "/health", http.StatusOK},

		// Shop pages
		{"Shop listing", "GET", "/shop", http.StatusOK},
		{"Premium shop", "GET", "/shop/premium", http.StatusOK},

		// Cart
		{"Cart page", "GET", "/cart", http.StatusOK},

		// Static pages
		{"About page", "GET", "/about", http.StatusOK},
		{"Contact page", "GET", "/contact", http.StatusOK},
		{"Portfolio page", "GET", "/portfolio", http.StatusOK},

		// Legal pages
		{"Privacy policy", "GET", "/privacy", http.StatusOK},
		{"Terms of service", "GET", "/terms", http.StatusOK},
		{"Shipping policy", "GET", "/shipping", http.StatusOK},

		// Auth pages
		{"Login page", "GET", "/login", http.StatusOK},
		{"Signup page", "GET", "/signup", http.StatusOK},
		{"Signup page alt", "GET", "/sign-up", http.StatusOK},

		// Custom quote
		{"Custom quote page", "GET", "/custom", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code,
				"Route %s %s should return %d, got %d",
				tt.method, tt.path, tt.wantStatus, rec.Code)
		})
	}
}

// TestTier2_AuthProtectedRoutes tests that auth-protected routes require authentication
func TestTier2_AuthProtectedRoutes(t *testing.T) {
	e, _ := setupTestEcho(t)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int // Without auth, should redirect to login or return 401
	}{
		// Account routes - should require auth
		{"Account dashboard", "GET", "/account", http.StatusFound}, // Redirects to /login
		{"Email preferences (new path)", "GET", "/account/email-preferences", http.StatusFound},
		{"Email preferences (redirect)", "GET", "/email-preferences", http.StatusMovedPermanently},

		// Admin routes - should require auth AND admin role
		{"Admin dashboard", "GET", "/admin", http.StatusFound},
		{"Admin products", "GET", "/admin/products", http.StatusFound},
		{"Admin orders", "GET", "/admin/orders", http.StatusFound},
		{"Admin contacts", "GET", "/admin/contacts", http.StatusFound},
		{"Admin abandoned carts", "GET", "/admin/abandoned-carts", http.StatusFound},
		{"Admin emails", "GET", "/admin/emails", http.StatusFound},
		{"Admin promotions", "GET", "/admin/promotions", http.StatusFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// These routes should either redirect or return 401, not return 200
			assert.NotEqual(t, http.StatusOK, rec.Code,
				"Protected route %s %s should not return 200 without auth",
				tt.method, tt.path)

			// Check it matches expected status
			assert.Equal(t, tt.wantStatus, rec.Code,
				"Protected route %s %s should return %d, got %d",
				tt.method, tt.path, tt.wantStatus, rec.Code)
		})
	}
}

// TestTier3_APIRoutes tests that API endpoints exist
func TestTier3_APIRoutes(t *testing.T) {
	e, _ := setupTestEcho(t)

	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusOK   bool   // If true, expect 200. If false, expect anything but 404
		allowedStatuses []int // Status codes that are acceptable
	}{
		// Cart API - These may return various statuses depending on session/data
		{"Get cart", "GET", "/api/cart", false, []int{http.StatusOK, http.StatusUnauthorized, http.StatusFound}},

		// Email preferences API
		{"Get email preferences", "GET", "/api/email-preferences?email=test@example.com", false,
			[]int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized}},

		// Promotions API
		{"Get popup status", "GET", "/api/promotions/popup-status", false,
			[]int{http.StatusOK, http.StatusBadRequest}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// Main assertion: route exists (not 404)
			assert.NotEqual(t, http.StatusNotFound, rec.Code,
				"API route %s %s should exist (not return 404)",
				tt.method, tt.path)

			// If wantStatusOK, expect 200
			if tt.wantStatusOK {
				assert.Equal(t, http.StatusOK, rec.Code,
					"API route %s %s should return 200, got %d",
					tt.method, tt.path, rec.Code)
			} else if len(tt.allowedStatuses) > 0 {
				// Check status is in allowed list
				assert.Contains(t, tt.allowedStatuses, rec.Code,
					"API route %s %s returned %d, expected one of %v",
					tt.method, tt.path, rec.Code, tt.allowedStatuses)
			}
		})
	}
}

// TestEmailPreferencesRedirect specifically tests the redirect we just added
func TestEmailPreferencesRedirect(t *testing.T) {
	e, _ := setupTestEcho(t)

	req := httptest.NewRequest("GET", "/email-preferences", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 301 Moved Permanently
	assert.Equal(t, http.StatusMovedPermanently, rec.Code,
		"Route /email-preferences should return 301")

	// Should redirect to /account/email-preferences
	location := rec.Header().Get("Location")
	assert.Equal(t, "/account/email-preferences", location,
		"Route /email-preferences should redirect to /account/email-preferences")
}

// TestNonExistentRoute verifies that truly non-existent routes return 404
func TestNonExistentRoute(t *testing.T) {
	e, _ := setupTestEcho(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"Random path", "GET", "/this-route-does-not-exist"},
		{"Random API path", "GET", "/api/nonexistent"},
		{"Random admin path", "GET", "/admin/fake-page"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code,
				"Non-existent route %s %s should return 404",
				tt.method, tt.path)
		})
	}
}

// TestStaticFiles tests that the public static file route is registered
func TestStaticFiles(t *testing.T) {
	e, _ := setupTestEcho(t)

	// Test that /public/* route exists (even if file doesn't exist, it shouldn't 404 on routing)
	req := httptest.NewRequest("GET", "/public/test.css", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Either 200 (file exists), 404 (file not found), or 403 (permission)
	// But should NOT be routing 404 (which would be "Not Found" from Echo)
	assert.NotEqual(t, http.StatusNotFound, rec.Code,
		"Static file route should be registered (file might not exist, but route should)")
}
