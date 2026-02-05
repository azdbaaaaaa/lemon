package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"lemon/internal/pkg/ctxutil"
	"lemon/internal/pkg/jwt"
)

// Auth JWT 认证中间件
// 从 Authorization header 中提取 Bearer token，验证后注入 user_id 到 context
func Auth(jwtUtil *jwt.JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 获取 Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40101,
				"message": "未授权",
			})
			c.Abort()
			return
		}

		// 提取 Token（Bearer {token}）
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40101,
				"message": "Invalid authorization header",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证 Token
		claims, err := jwtUtil.ValidateToken(tokenString)
		if err != nil {
			errorCode := 40102
			if err == jwt.ErrExpiredToken {
				errorCode = 40102
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    errorCode,
				"message": "Token无效或已过期",
			})
			c.Abort()
			return
		}

		// 将 user_id 注入到 context
		ctx := ctxutil.WithUserID(c.Request.Context(), claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

