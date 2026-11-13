package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	logger "GuiFernandess7/gothenticate/internal/infra/log"
	pkg "GuiFernandess7/gothenticate/internal/pkg"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

const tokenTTL = 24 * time.Hour

type AuthHandler struct {
    Repo      UserRepositoryInterface
    Log       logger.Logger
    JWTSecret []byte
    RDB       *redis.Client
}

func HealthRoute(c echo.Context) error {
    return c.JSON(http.StatusAccepted, map[string]string{
		"message": "OK.",
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	// 1. Clean body and Extract payload information
	var requestBody LoginRequest
	if c.Request().ContentLength != 0 {
		if err := c.Bind(&requestBody); err != nil {
			h.Log.Errorf("Failed to bind login payload: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": "invalid payload",
			})
		}
	}
	// 3. Find user in database
	h.Log.Infof("[LOGIN] Searching for user in database...")
	user, err := h.Repo.FindByEmail(c.Request().Context(), requestBody.Email)

	// 4. Always run the bcrypt compare, even if no user was found
	hash := pkg.DummyHash
	if err == nil && user != nil {
		hash = user.Password
	}
	passwordOK := pkg.CheckPassword(requestBody.Password, hash)

	if user == nil {
		h.Log.Infof("[LOGIN] User not found in database...")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"message": "invalid email or password",
		})
	}

	// 5. Check if the account is allowed to try logging in at all
	accountState := NewAccountState(user.ID, AccountStatus(user.Status), user.Retries)
	h.Log.Infof("[LOGIN] Checking user account state.")
	if err := accountState.CanAttemptLogin(); err != nil {
		h.Log.Errorf("[LOGIN] Error - User state: %v", err.Error())
		return c.JSON(http.StatusForbidden, map[string]string{
			"message": "something went wrong.",
		})
	}

	// 6. Record the attempt's outcome and persist the resulting state
	ctx := c.Request().Context()

	h.Log.Infof("[LOGIN] Verifying user password.")
	if !passwordOK {
		accountState.RecordFailedLogin()
		if err := h.Repo.UpdateAccountState(ctx, user.ID, accountState.Label(), accountState.Retries); err != nil {
			h.Log.Errorf("[LOGIN] Error logging in: %v", err.Error())
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"message": "something went wrong.",
			})
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"message": "invalid email or password",
		})
	}

	accountState.RecordSuccessfulLogin()
	if err := h.Repo.UpdateAccountState(ctx, user.ID, accountState.Label(), accountState.Retries); err != nil {
		h.Log.Errorf("[LOGIN] Error logging in: %v", err.Error())
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "something went wrong.",
		})
	}

	token, err := pkg.GenerateToken(h.JWTSecret, user.ID, user.Email, tokenTTL)
	if err != nil {
		h.Log.Errorf("[LOGIN] Failed to generate token for user %d: %v", user.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "something went wrong.",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Login Successfull.",
		"token":   token,
	})
}

func (h *AuthHandler) Signup(c echo.Context) error {
	// 1. Check payload information
	var requestBody SignUpRequest
	if c.Request().ContentLength != 0 {
		if err := c.Bind(&requestBody); err != nil {
			h.Log.Errorf("[SIGNUP] Failed to bind signup payload: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": "invalid payload",
			})
		}
	}

	h.Log.Infof("[SIGNUP] Received signup request for email %q.", requestBody.Email)

	// 2. Validate email, name and password
	if !pkg.IsEmailValid(requestBody.Email) {
		h.Log.Errorf("[SIGNUP] Invalid email: %q.", requestBody.Email)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid email",
		})
	}

	if !pkg.IsNameValid(requestBody.Name) {
		h.Log.Errorf("[SIGNUP] Invalid name: %q.", requestBody.Name)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid name",
		})
	}

	validPassword, errs := pkg.IsPasswordStrong(requestBody.Password)
	if errs != nil && !validPassword{
		h.Log.Errorf("[SIGNUP] Weak password for email %q: %v", requestBody.Email, errs)
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"errors": errs,
		})
	}

	// 3. Check for existing email
	existingUser, err := h.Repo.FindByEmail(c.Request().Context(), requestBody.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.Log.Errorf("[SIGNUP] Failed to fetch user in database %q: %v", requestBody.Email, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Something went wrong.",
		})
	}
	if existingUser != nil {
		h.Log.Errorf("[SIGNUP] Email already registered: %q", requestBody.Email)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "invalid email or password",
		})
	}

	// 4. Mount object
	hashedPassword, err := pkg.HashPassword(requestBody.Password)
	if err != nil {
		h.Log.Errorf("[SIGNUP] Failed to hash password for email %q: %v", requestBody.Email, err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "Something went wrong.",
		})
	}

	userObj := User{
		Name: requestBody.Name,
		Email: requestBody.Email,
		Password: hashedPassword,
	}

	// 5. Save in the database
	h.Log.Infof("[SIGNUP] Creating user for email %q.", requestBody.Email)
	err = h.Repo.Create(c.Request().Context(), &userObj)
	if err != nil {
		h.Log.Errorf("[SIGNUP] Failed to create user for email %q: %v", requestBody.Email, err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "Something went wrong.",
		})
	}

	h.Log.Infof("[SIGNUP] User created successfully with id %d.", userObj.ID)
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Signup Successfull.",
	})
}