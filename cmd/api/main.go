package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	database "GuiFernandess7/gothenticate/internal/infra/database"
	server "GuiFernandess7/gothenticate/internal/infra/server"

	"github.com/joho/godotenv"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)


func main() {
    err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

    sql_dsn := os.Getenv("SQLITE_DSN")
    db, err := database.Connect(sql_dsn)
	if err != nil {
		log.Error("Failed to connect to the database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping the database: %v", err)
	}

    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        log.Fatalf("JWT_SECRET environment variable is required")
    }

    redisConnectionString := os.Getenv("REDIS_CONN_STRING")
    if redisConnectionString == "" {
        log.Fatalf("JWT_SECRET environment variable is required")
    }

    opt, err := redis.ParseURL(redisConnectionString)
    if err != nil {
        log.Fatalf("REDIS_CONN_STRING enviroment variable is required.")
    }

    rdb := redis.NewClient(opt)
    srv := server.NewServer(db, []byte(jwtSecret), rdb)
    httpServer := &http.Server{
        Addr:    ":8080",
        Handler: srv.Router,
    }

    go func() {
        srv.Log.Info("Starting running on port 8080.")

        if err := httpServer.ListenAndServe(); err != nil &&
            err != http.ErrServerClosed {
            srv.Log.Error(err.Error())
        }
    }()

    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop

    ctx, cancel := context.WithTimeout(
        context.Background(),
        10*time.Second,
    )
    defer cancel()

    if err := httpServer.Shutdown(ctx); err != nil {
        srv.Log.Error("Error shutting down server")

        httpServer.Close()
    }

    srv.Log.Info("Servidor encerrado")
}