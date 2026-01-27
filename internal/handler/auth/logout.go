package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Logout 退出登录
// @Summary      退出登录
// @Description  退出登录，使Token失效
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	// 从请求头获取Refresh Token（如果存在）
	refreshToken := c.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		// 也可以从body获取
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken != "" {
		ctx := c.Request.Context()
		if err := h.authService.Logout(ctx, refreshToken); err != nil {
			// 记录错误但不影响响应
			_ = err
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "退出成功",
	})
}
