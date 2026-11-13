package server

import (
	logger "GuiFernandess7/gothenticate/internal/infra/log"
	customvalidator "GuiFernandess7/gothenticate/internal/infra/validator"
	"GuiFernandess7/gothenticate/internal/middleware"
	"database/sql"

	auth "GuiFernandess7/gothenticate/internal/auth"
	order "GuiFernandess7/gothenticate/internal/modules/order"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type application struct {
    Router    *echo.Echo
    Log       *logger.ConsoleLogger
    DB        *sql.DB
    RDB       *redis.Client
    JWTSecret []byte
}

func (s *application) initRoutes(db *sql.DB) {
    s.Router.GET("/health", auth.HealthRoute)

    // ========== AUTH ROUTES ================
	repo := auth.NewUserRepository(db)
	h := &auth.AuthHandler{Repo: repo, Log: s.Log, JWTSecret: s.JWTSecret, RDB: s.RDB}
    rateLimiter := middleware.NewRateLimiter(5, 10)
    jwtAuth := middleware.JWTAuth(s.JWTSecret)

    s.Router.POST("/login", h.Login)
    s.Router.POST("/signup", h.Signup, rateLimiter.Middleware)

    s.Router.GET("/me", func(c echo.Context) error {
        return c.JSON(200, map[string]interface{}{
            "user_id": c.Get(middleware.ContextKeyUserID),
            "email":   c.Get(middleware.ContextKeyEmail),
        })
    }, jwtAuth)

    // ========== ORDER ROUTES ================
    orderRepo := order.NewOrderRepository(db)
	o := &order.OrderHandler{Repo: *orderRepo, Log: s.Log, JWTSecret: s.JWTSecret, RDB: s.RDB}

    s.Router.GET("/orders", o.GetOrders, jwtAuth)
    s.Router.POST("/orders", o.AddOrder, jwtAuth)
    s.Router.GET("/orders/:id", o.GetOrderByID, jwtAuth)
    s.Router.PATCH("/orders/:id", o.UpdateOrderStatus, jwtAuth)
    s.Router.DELETE("/orders/:id", o.DeleteOrder, jwtAuth)
}

func NewServer(db *sql.DB,
            jwtSecret []byte,
            rdb *redis.Client,
    ) *application {
    e := echo.New()
    log := logger.NewLogger()

    e.Validator = customvalidator.New()
    e.Use(middleware.SecurityMiddleware)
    s := &application{
        Router:    e,
        Log:       log,
        DB:        db,
        RDB:       rdb,
        JWTSecret: jwtSecret,
    }
    s.initRoutes(db)
    return s
}