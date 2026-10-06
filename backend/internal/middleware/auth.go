package middleware

import (
	"net/http"
	"strings"

	"funchat/backend/pkg/jwt"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// Auth JWT 鉴权中间件
func Auth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "缺少认证信息")
			c.Abort()
			return
		}

		// Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Unauthorized(c, "认证格式错误")
			c.Abort()
			return
		}

		claims, err := jwtManager.ParseToken(parts[1])
		if err != nil {
			response.Unauthorized(c, "Token 无效或已过期")
			c.Abort()
			return
		}

		// 将用户信息写入上下文
		c.Set("user_id", claims.UserID)
		c.Set("phone", claims.Phone)
		c.Next()
	}
}

// CORSMiddleware 跨域中间件
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; !ok && len(allowed) > 0 {
				response.Forbidden(c, "origin is not allowed")
				c.Abort()
				return
			}
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimiter 简易频控中间件
func RateLimiter() gin.HandlerFunc {
	// V1: 基于 IP 的简易频控，后续接入 Redis 实现分布式频控
	return func(c *gin.Context) {
		// TODO: 接入 Redis 实现分布式频控
		c.Next()
	}
}
