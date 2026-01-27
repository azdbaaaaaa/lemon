package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetMe 获取当前用户信息
// @Summary      获取当前用户信息
// @Description  获取当前登录用户的详细信息
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/auth/me [get]
func (h *Handler) GetMe(c *gin.Context) {
	// 从请求头获取Token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    40101,
			Message: "未授权",
		})
		return
	}

	// 提取Token（Bearer {token}）
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    40101,
			Message: "Invalid authorization header",
		})
		return
	}

	tokenString := parts[1]
	user, err := h.authService.ValidateToken(tokenString)
	if err != nil {
		code := http.StatusUnauthorized
		errorCode := 40102

		if err.Error() == "Token已过期" {
			errorCode = 40102
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	// 转换为UserInfo
	userInfo := toUserInfo(user)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    userInfo,
	})
}
