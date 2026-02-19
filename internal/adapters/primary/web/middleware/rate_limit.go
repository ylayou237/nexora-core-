package middleware

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const rateLimitKeyFormat = "ratelimit:ip:%s"

type RateLimiter struct {
	client redis.UniversalClient
	limit  int64
	window time.Duration
}

func NewRateLimiter(client redis.UniversalClient, limit int64, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func extractIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ip := extractIP(r)
		key := fmt.Sprintf(rateLimitKeyFormat, ip)

		// Pipeline atomique pour INCR + EXPIRE
		pipe := rl.client.TxPipeline()
		incr := pipe.Incr(ctx, key)
		pipe.PTTL(ctx, key) // TTL actuel pour info
		_, err := pipe.Exec(ctx)
		if err != nil {
			log.Printf("⚠️ [RateLimiter] Redis error, request allowed: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		attempts := incr.Val()

		// Si première requête ou TTL expiré, remettre le TTL
		ttl, err := rl.client.TTL(ctx, key).Result()
		if err != nil || ttl <= 0 {
			rl.client.Expire(ctx, key, rl.window)
			ttl = rl.window
		}

		// Vérification de la limite
		if attempts > rl.limit {
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			http.Error(w, "429 Too Many Requests - Slow down!", http.StatusTooManyRequests)
			return
		}

		// Headers standard pour info
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rl.limit-attempts))
		next.ServeHTTP(w, r)
	})
}
