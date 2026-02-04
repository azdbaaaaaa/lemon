package resource

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"lemon/internal/service"
)

// GetDownloadURLRequest 获取下载URL请求
type GetDownloadURLRequest struct {
	ResourceID string `uri:"resource_id" binding:"required"` // 资源ID（必填）
	ExpiresIn  int    `form:"expires_in"`                    // 过期时间（秒，可选，默认3600）
}

// GetDownloadURLResponseData 获取下载URL响应数据
type GetDownloadURLResponseData struct {
	ResourceID  string `json:"resource_id"`  // 资源ID
	DownloadURL string `json:"download_url"` // 下载URL
	ExpiresAt   string `json:"expires_at"`    // 过期时间
	FileName    string `json:"file_name"`    // 文件名
	FileSize    int64  `json:"file_size"`    // 文件大小
	ContentType string `json:"content_type"` // MIME类型
}

// GetDownloadURL 获取下载URL（预签名URL）
// @Summary      获取下载URL
// @Description  根据资源ID获取预签名的下载URL，适用于客户端直接下载
// @Tags         资源管理
// @Accept       json
// @Produce      json
// @Param        resource_id  path      string  true   "资源ID"
// @Param        expires_in   query     int     false  "过期时间（秒，默认3600）"
// @Success      200          {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"resource_id\": \"...\", \"download_url\": \"...\", \"expires_at\": \"...\", \"file_name\": \"...\", \"file_size\": 1024, \"content_type\": \"...\"}}"
// @Failure      400          {object}  ErrorResponse  "请求参数错误"
// @Failure      404          {object}  ErrorResponse  "资源不存在"
// @Failure      500          {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/resources/{resource_id}/download-url [get]
func (h *Handler) GetDownloadURL(c *gin.Context) {
	var req GetDownloadURLRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid resource_id",
			Detail:  err.Error(),
		})
		return
	}

	// 解析 expires_in 参数
	expiresIn := time.Hour // 默认1小时
	if expiresInStr := c.Query("expires_in"); expiresInStr != "" {
		if seconds, err := strconv.Atoi(expiresInStr); err == nil && seconds > 0 {
			expiresIn = time.Duration(seconds) * time.Second
		}
	}

	ctx := c.Request.Context()

	// TODO: 从认证中间件中获取用户ID
	// 目前先使用空字符串，视为系统内部请求
	userID := ""

	// 调用Service层
	result, err := h.resourceService.GetDownloadURL(ctx, &service.GetDownloadURLRequest{
		UserID:     userID,
		ResourceID: req.ResourceID,
		ExpiresIn:  expiresIn,
	})
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "资源不存在" {
			code = http.StatusNotFound
			errorCode = 40401
		} else if err.Error() == "无权访问该资源" {
			code = http.StatusForbidden
			errorCode = 40301
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
		"data": GetDownloadURLResponseData{
			ResourceID:  result.ResourceID,
			DownloadURL: result.DownloadURL,
			ExpiresAt:   result.ExpiresAt.Format(time.RFC3339),
			FileName:    result.FileName,
			FileSize:    result.FileSize,
			ContentType: result.ContentType,
		},
	})
}
