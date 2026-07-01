package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	gatewayMiddleware "github.com/snaply/api-gateway/internal/middleware"
	"github.com/snaply/api-gateway/internal/config"
	"github.com/snaply/api-gateway/internal/proxy"
	"github.com/snaply/api-gateway/internal/ws"
	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	p, err := proxy.New(cfg.Upstreams.UserServiceURL, cfg.Upstreams.PostServiceURL, cfg.Upstreams.RelationServiceURL, cfg.Upstreams.LikeServiceURL, log)
	if err != nil {
		log.Fatal("failed to create proxy", zap.Error(err))
	}

	hub := ws.NewHub(log)
	authMW := gatewayMiddleware.NewAuthMiddleware(cfg.JWT.Secret, log)
	rateMW := gatewayMiddleware.NewRateLimiter(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.Burst, log)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(rateMW.Limit)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// Public auth routes — no JWT required
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Handle("/*", p.UserService())
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMW.Authenticate)

		r.Handle("/api/v1/users/{user_id}/posts", p.PostService())
		r.Handle("/api/v1/users/{user_id}/posts/count", p.PostService())
		r.Handle("/api/v1/users/*", p.UserService())
		r.Handle("/api/v1/posts", p.PostService())
		r.Handle("/api/v1/posts/*", p.PostService())
		r.Handle("/api/v1/comments/*", p.PostService())
		r.Handle("/api/v1/feed", p.PostService())
		r.Handle("/api/v1/feed/*", p.PostService())
		r.Handle("/api/v1/relations/*", p.RelationService())
		r.Handle("/api/v1/likes/*", p.LikeService())

		// WebSocket endpoint
		r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(gatewayMiddleware.UserIDKey).(uuid.UUID)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			hub.ServeWS(w, r, userID)
		})
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info("api-gateway starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
