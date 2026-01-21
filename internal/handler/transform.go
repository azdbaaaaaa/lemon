package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lemon/internal/model"
	"lemon/internal/service"
)

// TransformHandler 文本转换处理器
type TransformHandler struct {
	transformSvc *service.TransformService
}

// NewTransformHandler 创建文本转换处理器
func NewTransformHandler(transformSvc *service.TransformService) *TransformHandler {
	return &TransformHandler{
		transformSvc: transformSvc,
	}
}

// Transform 文本转换接口
// POST /api/v1/transform
// Request: { "text": "原始文本", "prompt": "翻译成英文" }
// Response: { "text": "Translated text", "usage": {...} }
func (h *TransformHandler) Transform(c *gin.Context) {
	var req model.TransformRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	resp, err := h.transformSvc.Transform(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    50001,
			Message: "Transform failed",
			Detail:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
