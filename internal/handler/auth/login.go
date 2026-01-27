package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LoginRequest 用户登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名（必填）
	Password string `json:"password" binding:"required"` // 密码（必填）
}

// LoginResponseData 登录响应数据
type LoginResponseData struct {
	AccessToken  string   `json:"access_token"`  // Access Token
	RefreshToken string   `json:"refresh_token"` // Refresh Token
	ExpiresIn    int      `json:"expires_in"`    // 过期时间（秒）
	TokenType    string   `json:"token_type"`    // Token类型：Bearer
	User         UserInfo `json:"user"`          // 用户信息
}

// UserInfo 用户信息（用于响应，所有API共用）
type UserInfo struct {
	ID          string       `json:"id"`                      // 用户ID
	Username    string       `json:"username"`                // 用户名
	Email       string       `json:"email"`                   // 邮箱
	Role        string       `json:"role"`                    // 角色：admin/editor/reviewer
	Status      string       `json:"status"`                  // 状态：active/inactive/banned
	Profile     *UserProfile `json:"profile,omitempty"`       // 用户资料
	LastLoginAt string       `json:"last_login_at,omitempty"` // 最后登录时间
	CreatedAt   string       `json:"created_at,omitempty"`    // 创建时间
}

// UserProfile 用户资料（所有API共用）
type UserProfile struct {
	Nickname string `json:"nickname,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Phone    string `json:"phone,omitempty"`
}

// Login 用户登录
// @Summary      用户登录
// @Description  用户登录，返回Access Token和Refresh Token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "登录请求"
// @Success      200     {object}  map[string]interface{}
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.authService.Login(ctx, req.Username, req.Password)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch err.Error() {
		case "用户不存在":
			code = http.StatusUnauthorized
			errorCode = 40101
		case "密码错误":
			code = http.StatusUnauthorized
			errorCode = 40101
		case "用户未激活，请联系管理员":
			code = http.StatusForbidden
			errorCode = 40005
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

	// 将Service返回的User转换为UserInfo
	userInfo := toUserInfo(resp.User)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "登录成功",
		"data": LoginResponseData{
			AccessToken:  resp.AccessToken,
			RefreshToken: resp.RefreshToken,
			ExpiresIn:    resp.ExpiresIn,
			TokenType:    resp.TokenType,
			User:         userInfo,
		},
	})
}
