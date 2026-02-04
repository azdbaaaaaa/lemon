package resource

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"lemon/internal/service"
)

// GetResourceRequest 获取资源请求
type GetResourceRequest struct {
	ResourceID string `uri:"resource_id" binding:"required"` // 资源ID（必填）
}

// GetResourceResponseData 获取资源响应数据
type GetResourceResponseData struct {
	Resource ResourceInfo `json:"resource"` // 资源信息
}

// GetResource 获取资源信息
// @Summary      获取资源信息
// @Description  根据资源ID获取资源的详细信息（元数据，不包含文件内容）
// @Tags         资源管理
// @Accept       json
// @Produce      json
// @Param        resource_id  path      string  true  "资源ID"
// @Success      200          {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"resource\": {...}}}"
// @Failure      400          {object}  ErrorResponse  "请求参数错误"
// @Failure      404          {object}  ErrorResponse  "资源不存在"
// @Failure      500          {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/resources/{resource_id} [get]
func (h *Handler) GetResource(c *gin.Context) {
	var req GetResourceRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid resource_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// TODO: 从认证中间件中获取用户ID
	// 目前先使用空字符串，视为系统内部请求
	userID := ""

	// 调用Service层
	result, err := h.resourceService.GetResource(ctx, &service.GetResourceRequest{
		UserID:     userID,
		ResourceID: req.ResourceID,
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
		"data": GetResourceResponseData{
			Resource: toResourceInfo(result.Resource),
		},
	})
}
