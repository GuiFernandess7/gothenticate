package auth_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	auth "GuiFernandess7/gothenticate/internal/auth"
	pkg "GuiFernandess7/gothenticate/internal/pkg"

	"github.com/labstack/echo/v4"
)

// ---------- test doubles ----------

type mockUserRepository struct {
	findByEmail        func(ctx context.Context, email string) (*auth.User, error)
	create             func(ctx context.Context, user *auth.User) error
	updateAccountState func(ctx context.Context, userID int64, status auth.AccountStatus, retries int) error
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	return m.findByEmail(ctx, email)
}

func (m *mockUserRepository) Create(ctx context.Context, user *auth.User) error {
	return m.create(ctx, user)
}

func (m *mockUserRepository) UpdateAccountState(ctx context.Context, userID int64, status auth.AccountStatus, retries int) error {
	return m.updateAccountState(ctx, userID, status, retries)
}

type noopLogger struct{}

func (noopLogger) Info(string)           {}
func (noopLogger) Warn(string)           {}
func (noopLogger) Error(string)          {}
func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Warnf(string, ...any)  {}
func (noopLogger) Errorf(string, ...any) {}

func newJSONContext(method, target string, body []byte) (*httptest.ResponseRecorder, echo.Context) {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e := echo.New()
	return rec, e.NewContext(req, rec)
}

func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := pkg.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return hash
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}
	return out
}

// ---------- Login ----------

func TestLogin_InvalidPayload(t *testing.T) {
	rec, c := newJSONContext(http.MethodPost, "/login", []byte("{invalid-json"))
	h := &auth.AuthHandler{Log: noopLogger{}}

	if err := h.Login(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "invalid payload" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	payload, _ := json.Marshal(auth.LoginRequest{Email: "missing@example.com", Password: "whatever"})
	rec, c := newJSONContext(http.MethodPost, "/login", payload)

	repo := &mockUserRepository{
		findByEmail: func(ctx context.Context, email string) (*auth.User, error) {
			return nil, sql.ErrNoRows
		},
	}
	h := &auth.AuthHandler{Repo: repo, Log: noopLogger{}}

	if err := h.Login(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "invalid email or password" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	payload, _ := json.Marshal(auth.LoginRequest{Email: "user@example.com", Password: "WrongPass1!"})
	rec, c := newJSONContext(http.MethodPost, "/login", payload)

	updateCalled := false
	repo := &mockUserRepository{
		findByEmail: func(ctx context.Context, email string) (*auth.User, error) {
			return &auth.User{
				ID:       1,
				Email:    "user@example.com",
				Password: mustHash(t, "CorrectPass1!"),
				Status:   string(auth.StatusLabelActive),
				Retries:  0,
			}, nil
		},
		updateAccountState: func(ctx context.Context, userID int64, status auth.AccountStatus, retries int) error {
			updateCalled = true
			if retries != 1 {
				t.Errorf("expected retries to be incremented to 1, got %d", retries)
			}
			return nil
		},
	}
	h := &auth.AuthHandler{Repo: repo, Log: noopLogger{}}

	if err := h.Login(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if !updateCalled {
		t.Fatal("expected UpdateAccountState to be called after a failed login")
	}
}

func TestLogin_Success(t *testing.T) {
	payload, _ := json.Marshal(auth.LoginRequest{Email: "user@example.com", Password: "CorrectPass1!"})
	rec, c := newJSONContext(http.MethodPost, "/login", payload)

	repo := &mockUserRepository{
		findByEmail: func(ctx context.Context, email string) (*auth.User, error) {
			return &auth.User{
				ID:       1,
				Email:    "user@example.com",
				Password: mustHash(t, "CorrectPass1!"),
				Status:   string(auth.StatusLabelActive),
				Retries:  3,
			}, nil
		},
		updateAccountState: func(ctx context.Context, userID int64, status auth.AccountStatus, retries int) error {
			if retries != 0 {
				t.Errorf("expected retries to reset to 0 on success, got %d", retries)
			}
			if status != auth.StatusLabelActive {
				t.Errorf("expected status to remain active, got %s", status)
			}
			return nil
		},
	}
	secret := []byte("test-secret")
	h := &auth.AuthHandler{Repo: repo, Log: noopLogger{}, JWTSecret: secret}

	if err := h.Login(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := decodeBody(t, rec)
	if body["message"] != "Login Successfull." {
		t.Fatalf("unexpected message: %v", body["message"])
	}

	tokenString, ok := body["token"].(string)
	if !ok || tokenString == "" {
		t.Fatalf("expected a non-empty token in the response, got: %v", body["token"])
	}

	claims, err := pkg.ParseToken(secret, tokenString)
	if err != nil {
		t.Fatalf("returned token should be valid, got error: %v", err)
	}
	if claims.UserID != 1 || claims.Email != "user@example.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestLogin_AccountLocked(t *testing.T) {
	payload, _ := json.Marshal(auth.LoginRequest{Email: "user@example.com", Password: "CorrectPass1!"})
	rec, c := newJSONContext(http.MethodPost, "/login", payload)

	updateCalled := false
	repo := &mockUserRepository{
		findByEmail: func(ctx context.Context, email string) (*auth.User, error) {
			return &auth.User{
				ID:       1,
				Email:    "user@example.com",
				Password: mustHash(t, "CorrectPass1!"),
				Status:   string(auth.StatusLabelLocked),
				Retries:  5,
			}, nil
		},
		updateAccountState: func(ctx context.Context, userID int64, status auth.AccountStatus, retries int) error {
			updateCalled = true
			return nil
		},
	}
	h := &auth.AuthHandler{Repo: repo, Log: noopLogger{}}

	if err := h.Login(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if updateCalled {
		t.Fatal("UpdateAccountState should not be called when the account is locked")
	}
}

func TestLogin_UpdateAccountStateError(t *testing.T) {
	payload, _ := json.Marshal(auth.LoginRequest{Email: "user@example.com", Password: "CorrectPass1!"})
	rec, c := newJSONContext(http.MethodPost, "/login", payload)

	repo := &mockUserRepository{
		findByEmail: func(ctx context.Context, email string) (*auth.User, error) {
			return &auth.User{
				ID:       1,
				Email:    "user@example.com",
				Password: mustHash(t, "CorrectPass1!"),
				Status:   string(auth.StatusLabelActive),
				Retries:  0,
			}, nil
		},
		updateAccountState: func(ctx context.Context, userID int64, status auth.AccountStatus, retries int) error {
			return sql.ErrConnDone
		},
	}
	h := &auth.AuthHandler{Repo: repo, Log: noopLogger{}}

	if err := h.Login(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// ---------- Signup ----------

func TestSignup_InvalidPayload(t *testing.T) {
	rec, c := newJSONContext(http.MethodPost, "/signup", []byte("{invalid-json"))
	h := &auth.AuthHandler{Log: noopLogger{}}

	if err := h.Signup(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "invalid payload" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestSignup_InvalidEmail(t *testing.T) {
	payload, _ := json.Marshal(auth.SignUpRequest{Name: "John Doe", Email: "not-an-email", Password: "StrongPass1!"})
	rec, c := newJSONContext(http.MethodPost, "/signup", payload)
	h := &auth.AuthHandler{Log: noopLogger{}}

	if err := h.Signup(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "invalid email" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestSignup_InvalidName(t *testing.T) {
	payload, _ := json.Marshal(auth.SignUpRequest{Name: "John123", Email: "john@example.com", Password: "StrongPass1!"})
	rec, c := newJSONContext(http.MethodPost, "/signup", payload)
	h := &auth.AuthHandler{Log: noopLogger{}}

	if err := h.Signup(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "invalid name" {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestSignup_WeakPassword(t *testing.T) {
	payload, _ := json.Marshal(auth.SignUpRequest{Name: "John Doe", Email: "john@example.com", Password: "weak"})
	rec, c := newJSONContext(http.MethodPost, "/signup", payload)
	h := &auth.AuthHandler{Log: noopLogger{}}

	if err := h.Signup(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	body := decodeBody(t, rec)
	if _, ok := body["errors"]; !ok {
		t.Fatalf("expected an \"errors\" field listing password issues, got %v", body)
	}
}

func TestSignup_RepositoryError(t *testing.T) {
	payload, _ := json.Marshal(auth.SignUpRequest{Name: "John Doe", Email: "john@example.com", Password: "StrongPass1!"})
	rec, c := newJSONContext(http.MethodPost, "/signup", payload)

	repo := &mockUserRepository{
		findByEmail: func(ctx context.Context, email string) (*auth.User, error) {
			return nil, sql.ErrNoRows
		},
		create: func(ctx context.Context, user *auth.User) error {
			return sql.ErrConnDone
		},
	}
	h := &auth.AuthHandler{Repo: repo, Log: noopLogger{}}

	if err := h.Signup(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "Something went wrong." {
		t.Fatalf("unexpected message: %v", body["message"])
	}
}

func TestSignup_EmailAlreadyRegistered(t *testing.T) {
	payload, _ := json.Marshal(auth.SignUpRequest{Name: "John Doe", Email: "john@example.com", Password: "StrongPass1!"})
	rec, c := newJSONContext(http.MethodPost, "/signup", payload)

	createCalled := false
	repo := &mockUserRepository{
		findByEmail: func(ctx context.Context, email string) (*auth.User, error) {
			return &auth.User{ID: 1, Email: email}, nil
		},
		create: func(ctx context.Context, user *auth.User) error {
			createCalled = true
			return nil
		},
	}
	h := &auth.AuthHandler{Repo: repo, Log: noopLogger{}}

	if err := h.Signup(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "invalid email or password" {
		t.Fatalf("expected a generic message that does not leak account existence, got: %v", body["message"])
	}
	if createCalled {
		t.Fatal("Create should not be called when the email is already registered")
	}
}

func TestSignup_Success(t *testing.T) {
	payload, _ := json.Marshal(auth.SignUpRequest{Name: "John Doe", Email: "john@example.com", Password: "StrongPass1!"})
	rec, c := newJSONContext(http.MethodPost, "/signup", payload)

	var created *auth.User
	repo := &mockUserRepository{
		findByEmail: func(ctx context.Context, email string) (*auth.User, error) {
			return nil, sql.ErrNoRows
		},
		create: func(ctx context.Context, user *auth.User) error {
			created = user
			user.ID = 42
			return nil
		},
	}
	h := &auth.AuthHandler{Repo: repo, Log: noopLogger{}}

	if err := h.Signup(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := decodeBody(t, rec); body["message"] != "Signup Successfull." {
		t.Fatalf("unexpected message: %v", body["message"])
	}
	if created == nil {
		t.Fatal("expected Create to be called")
	}
	if created.Password == "StrongPass1!" {
		t.Fatal("expected the password to be hashed before being persisted")
	}
	if !pkg.CheckPassword("StrongPass1!", created.Password) {
		t.Fatal("stored hash does not match the original password")
	}
}
