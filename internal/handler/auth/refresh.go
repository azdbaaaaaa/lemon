package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"` // Refresh Token（必填）
}

// RefreshTokenResponseData 刷新Token响应数据
type RefreshTokenResponseData struct {
	AccessToken string `json:"access_token"` // Access Token
	ExpiresIn   int    `json:"expires_in"`   // 过期时间（秒）
	TokenType   string `json:"token_type"`   // Token类型：Bearer
}

// Refresh 刷新Token
// @Summary      刷新Token
// @Description  使用Refresh Token刷新Access Token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      RefreshTokenRequest  true  "刷新Token请求"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /api/v1/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch err.Error() {
		case "Token无效":
			code = http.StatusUnauthorized
			errorCode = 40102
		case "Token已过期":
			code = http.StatusUnauthorized
			errorCode = 40103
		case "用户不存在":
			code = http.StatusUnauthorized
			errorCode = 40101
		case "用户已被禁用":
			code = http.StatusForbidden
			errorCode = 40006
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": RefreshTokenResponseData{
			AccessToken: resp.AccessToken,
			ExpiresIn:   resp.ExpiresIn,
			TokenType:   resp.TokenType,
		},
	})
}
