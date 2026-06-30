package middleware

import (
	"net"
	"net/http"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*rate.Limiter
	rps     rate.Limit
	burst   int
	log     *zap.Logger
}

func NewRateLimiter(rps float64, burst int, log *zap.Logger) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*rate.Limiter),
		rps:     rate.Limit(rps),
		burst:   burst,
		log:     log,
	}
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if lim, ok := rl.clients[ip]; ok {
		return lim
	}
	lim := rate.NewLimiter(rl.rps, rl.burst)
	rl.clients[ip] = lim
	return lim
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		if !rl.getLimiter(ip).Allow() {
			respondError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}
