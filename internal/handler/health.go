package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health 健康检查
// @Summary      健康检查
// @Description  检查服务健康状态
// @Tags         健康检查
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// Ready 就绪检查
// @Summary      就绪检查
// @Description  检查服务是否就绪
// @Tags         健康检查
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
