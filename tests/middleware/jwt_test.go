package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"GuiFernandess7/gothenticate/internal/middleware"
	pkg "GuiFernandess7/gothenticate/internal/pkg"

	"github.com/labstack/echo/v4"
)

func newProtectedContext(authHeader string) (*httptest.ResponseRecorder, echo.Context) {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if authHeader != "" {
		req.Header.Set(echo.HeaderAuthorization, authHeader)
	}
	rec := httptest.NewRecorder()
	e := echo.New()
	return rec, e.NewContext(req, rec)
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	rec, c := newProtectedContext("")
	called := false
	handler := middleware.JWTAuth([]byte("secret"))(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if called {
		t.Fatal("next handler should not be called without a valid token")
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	rec, c := newProtectedContext("Bearer not-a-real-token")
	handler := middleware.JWTAuth([]byte("secret"))(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	secret := []byte("secret")
	token, err := pkg.GenerateToken(secret, 1, "user@example.com", -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	rec, c := newProtectedContext("Bearer " + token)
	handler := middleware.JWTAuth(secret)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	secret := []byte("secret")
	token, err := pkg.GenerateToken(secret, 42, "user@example.com", time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	rec, c := newProtectedContext("Bearer " + token)
	var gotUserID interface{}
	var gotEmail interface{}
	handler := middleware.JWTAuth(secret)(func(c echo.Context) error {
		gotUserID = c.Get(middleware.ContextKeyUserID)
		gotEmail = c.Get(middleware.ContextKeyEmail)
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if gotUserID != int64(42) {
		t.Fatalf("expected user_id 42 in context, got %v", gotUserID)
	}
	if gotEmail != "user@example.com" {
		t.Fatalf("expected email in context, got %v", gotEmail)
	}
}
