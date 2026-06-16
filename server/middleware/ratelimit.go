package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(addr, password string, db int) *RateLimiter {
	if addr == "" {
		return &RateLimiter{}
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		Protocol: 2,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		slog.Warn("Redis rate limiter unavailable, rate limiting disabled", "Error", err)
		return &RateLimiter{}
	}
	return &RateLimiter{client: rdb}
}

func (r *RateLimiter) checkLimit(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, error) {
	if r.client == nil {
		return true, 0, nil
	}
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		slog.Warn("Rate limiter Redis error", "Error", err)
		return true, 0, err
	}
	count := incr.Val()
	if count > limit {
		return false, count, nil
	}
	return true, count, nil
}

func (r *RateLimiter) RateLimitLogin() echo.MiddlewareFunc {
	return r.rateLimitMiddleware("login", 5, 15*time.Minute)
}

func (r *RateLimiter) RateLimitRegister() echo.MiddlewareFunc {
	return r.rateLimitMiddleware("register", 3, 15*time.Minute)
}

func (r *RateLimiter) RateLimitGeneral(postsPerMinute int64) echo.MiddlewareFunc {
	window := time.Minute
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip safe methods (page loads, WebSocket upgrades)
			method := c.Request().Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				return next(c)
			}

			// Skip server-to-server webhooks
			if c.Request().URL.Path == "/tbaWebhook" {
				return next(c)
			}

			// Use user UUID as key when authenticated; fall back to IP
			var key string
			userUuidVal := c.Get("userUuid")
			if userUuidVal != nil {
				key = fmt.Sprintf("rate_limit:general:%v", userUuidVal)
			} else {
				key = fmt.Sprintf("rate_limit:general:%s", c.RealIP())
			}

			allowed, _, err := r.checkLimit(c.Request().Context(), key, postsPerMinute, window)
			if err != nil {
				return next(c) // Fail open
			}
			if !allowed {
				c.Response().Header().Set("Retry-After", strconv.FormatInt(int64(window.Seconds()), 10))
				return c.NoContent(http.StatusTooManyRequests)
			}
			return next(c)
		}
	}
}

func (r *RateLimiter) rateLimitMiddleware(prefix string, limit int64, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			key := fmt.Sprintf("rate_limit:%s:%s", prefix, ip)
			allowed, _, err := r.checkLimit(c.Request().Context(), key, limit, window)
			if err != nil {
				// Fail open on Redis errors
				return next(c)
			}
			if !allowed {
				c.Response().Header().Set("Retry-After", strconv.FormatInt(int64(window.Seconds()), 10))
				return c.NoContent(http.StatusTooManyRequests)
			}
			return next(c)
		}
	}
}
