package middleware

import (
	"net/http"
	"sync"
	"time"

	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// MemoryRateLimiter is intentionally process-local. Its interface can be
// replaced by Redis without changing handlers; it still protects a single
// instance from basic SMS and credential abuse today.
func MemoryRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	type entry struct { count int; resetAt time.Time }
	var mu sync.Mutex
	entries := map[string]entry{}
	return func(c *gin.Context) {
		key, now := c.ClientIP(), time.Now()
		mu.Lock()
		item := entries[key]
		if item.resetAt.Before(now) { item = entry{resetAt: now.Add(window)} }
		item.count++
		entries[key] = item
		mu.Unlock()
		if item.count > limit {
			response.Error(c, http.StatusTooManyRequests, 42900, "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}
