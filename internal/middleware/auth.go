package middleware

import (
	"net"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
    mu      sync.Mutex
    clients map[string]*rate.Limiter
    limit   rate.Limit
    burst   int
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
    return &RateLimiter{
        clients: make(map[string]*rate.Limiter),
        limit:   rate.Limit(rps),
        burst:   burst,
    }
}

func (rl *RateLimiter) Middleware(
    next echo.HandlerFunc,
) echo.HandlerFunc {
    return func(c echo.Context) error {
        ip, _, err := net.SplitHostPort(c.Request().RemoteAddr)

        if err != nil {ip = c.RealIP()}

        rl.mu.Lock()
        limiter, exists := rl.clients[ip]

        if !exists {
            limiter = rate.NewLimiter(rl.limit, rl.burst)
            rl.clients[ip] = limiter
        }

        rl.mu.Unlock()

        if !limiter.Allow() {
            return c.JSON(
                http.StatusTooManyRequests,
                map[string]string{
                    "error": "too many requests",
                },
            )
        }

        return next(c)
    }
}


func SecurityMiddleware(
    next echo.HandlerFunc,
) echo.HandlerFunc {
    return func(c echo.Context) error {
        response := c.Response()
        response.Header().Set("X-Content-Type-Options", "nosniff")
        response.Header().Set("X-Frame-Options", "DENY")
        response.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        return next(c)
    }
}