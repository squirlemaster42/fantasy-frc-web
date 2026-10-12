package middleware

import (
	"context"
	"fmt"
	"net/http"
	"server/log"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
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
			Protocol: redisProtocolVersion,
		})
	ctx, cancel := context.WithTimeout(context.Background(), RateLimitRedisPingTimeout())
	defer cancel()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Error(ctx, "Redis rate limiter unavailable, rate limiting disabled", "error", err)
		return &RateLimiter{}
	}
	return &RateLimiter{client: rdb}
}

func (r *RateLimiter) checkLimit(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, time.Duration, error) {
	if r.client == nil {
		return true, 0, 0, nil
	}

	now := time.Now()
	windowStart := now.Truncate(window)
	windowKey := fmt.Sprintf("%s:%d", key, windowStart.Unix())

	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, windowKey)
	pipe.Expire(ctx, windowKey, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Error(ctx, "Rate limiter Redis error", "error", err)
		return true, 0, 0, fmt.Errorf("failed to execute rate limit pipeline: %w", err)
	}

	count := incr.Val()
	retryAfter := window - now.Sub(windowStart)
	if count > limit {
		return false, count, retryAfter, nil
	}
	return true, count, retryAfter, nil
}

func (r *RateLimiter) RateLimitLogin() echo.MiddlewareFunc {
	return r.rateLimitMiddleware(rateLimitKeyPrefixLogin, RateLimitLoginAttempts(), RateLimitAuthWindow())
}

func (r *RateLimiter) RateLimitRegister() echo.MiddlewareFunc {
	return r.rateLimitMiddleware(rateLimitKeyPrefixRegister, RateLimitRegisterAttempts(), RateLimitAuthWindow())
}

func (r *RateLimiter) RateLimitGeneral(postsPerMinute int64) echo.MiddlewareFunc {
	window := rateLimitGeneralWindow
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
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
				key = fmt.Sprintf("%s:%v", rateLimitKeyPrefixGeneral, userUuidVal)
			} else {
				key = fmt.Sprintf("%s:%s", rateLimitKeyPrefixGeneral, c.RealIP())
			}

			allowed, _, retryAfter, err := r.checkLimit(c.Request().Context(), key, postsPerMinute, window)
			if err != nil {
				log.Warn(c.Request().Context(), "Rate limiter failing open", "path", c.Request().URL.Path, "ip", c.RealIP(), "error", err)
				return next(c) // Fail open
			}
			if !allowed {
				log.Warn(c.Request().Context(), "Rate limit exceeded", "path", c.Request().URL.Path, "ip", c.RealIP(), "key", key)
				c.Response().Header().Set("Retry-After", strconv.FormatInt(int64(retryAfter.Round(time.Second).Seconds()), 10))
				return c.NoContent(http.StatusTooManyRequests)
			}
			return next(c)
		}
	}
}

func (r *RateLimiter) rateLimitMiddleware(prefix string, limit int64, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ip := c.RealIP()
			key := fmt.Sprintf("%s:%s", prefix, ip)
			allowed, _, retryAfter, err := r.checkLimit(c.Request().Context(), key, limit, window)
			if err != nil {
				log.Warn(c.Request().Context(), "Rate limiter failing open", "path", c.Request().URL.Path, "ip", ip, "prefix", prefix, "error", err)
				// Fail open on Redis errors
				return next(c)
			}
			if !allowed {
				log.Warn(c.Request().Context(), "Rate limit exceeded", "path", c.Request().URL.Path, "ip", ip, "prefix", prefix)
				c.Response().Header().Set("Retry-After", strconv.FormatInt(int64(retryAfter.Round(time.Second).Seconds()), 10))
				return c.NoContent(http.StatusTooManyRequests)
			}
			return next(c)
		}
	}
}
