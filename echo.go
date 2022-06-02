package ratelimit

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

// NewRateLimitMiddleware for echo middleware
func NewRateLimitMiddleware(rate, maxBurst int, duration time.Duration) echo.MiddlewareFunc {
	if rate == 0 {
		rate = 1
	}

	var rateLimiter = New(rate, maxBurst, duration)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !rateLimiter.Take() {
				resp := map[string]string{"message": "too many requests, rate limited"}
				return c.JSON(http.StatusTooManyRequests, resp)
			}

			return next(c)
		}
	}
}

// NewSpinRateLimitMiddleware for echo middleware
func NewSpinRateLimitMiddleware(rate, maxBurst int, duration, timeout time.Duration) echo.MiddlewareFunc {
	if rate == 0 {
		rate = 1
	}

	var rateLimiter = New(rate, maxBurst, duration)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !rateLimiter.SpinTake(timeout) {
				resp := map[string]string{"message": "too many requests, rate limited"}
				return c.JSON(http.StatusTooManyRequests, resp)
			}

			return next(c)
		}
	}
}
