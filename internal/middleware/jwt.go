package middleware

import (
	"net/http"
	"strings"

	pkg "GuiFernandess7/gothenticate/internal/pkg"

	"github.com/labstack/echo/v4"
)

const (
	ContextKeyUserID = "user_id"
	ContextKeyEmail  = "user_email"
)

func JWTAuth(secret []byte) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get(echo.HeaderAuthorization)

			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"message": "missing or invalid authorization header",
				})
			}

			tokenString := strings.TrimPrefix(authHeader, prefix)
			claims, err := pkg.ParseToken(secret, tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"message": "invalid or expired token",
				})
			}

			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyEmail, claims.Email)

			return next(c)
		}
	}
}
