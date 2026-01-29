package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"` // 用户名（必填，3-50字符）
	Email    string `json:"email" binding:"required,email"`           // 邮箱（必填，需符合邮箱格式）
	Password string `json:"password" binding:"required,min=6"`        // 密码（必填，至少6位）
	Nickname string `json:"nickname,omitempty"`                       // 昵称（可选）
}

// RegisterResponseData 注册响应数据
type RegisterResponseData struct {
	UserID   string `json:"user_id"`  // 用户ID
	Username string `json:"username"` // 用户名
	Status   string `json:"status"`   // 状态：inactive（新注册用户需要管理员审核）
}

// Register 用户注册
// @Summary      用户注册
// @Description  注册新用户，注册后状态为inactive，需要管理员审核
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "注册请求"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层（传递基本类型参数，不依赖Handler层的Request类型）
	resp, err := h.authService.Register(ctx, req.Username, req.Email, req.Password, req.Nickname)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch err.Error() {
		case "用户已存在":
			code = http.StatusBadRequest
			errorCode = 40001
		case "邮箱已被注册":
			code = http.StatusBadRequest
			errorCode = 40002
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "注册成功，等待管理员审核",
		"data": RegisterResponseData{
			UserID:   resp.UserID,
			Username: resp.Username,
			Status:   resp.Status,
		},
	})
}
