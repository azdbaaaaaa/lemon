package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lemon/internal/model/novel"
)

// GetVideosByStatusRequest 根据状态查询视频请求
type GetVideosByStatusRequest struct {
	Status string `form:"status" binding:"required,oneof=pending processing completed failed"` // 视频状态（必填）：pending, processing, completed, failed
}

// GetVideosByStatusResponseData 根据状态查询视频响应数据
type GetVideosByStatusResponseData struct {
	Videos []VideoInfo `json:"videos"` // 视频列表
	Count  int         `json:"count"`  // 视频数量
	Status string      `json:"status"` // 查询的状态
}

// GetVideosByStatus 根据状态查询视频（用于轮询）
// @Summary      根据状态查询视频
// @Description  根据状态查询视频列表，用于轮询视频生成进度。支持的状态：pending（待处理）、processing（处理中）、completed（已完成）、failed（失败）
// @Tags         视频查询
// @Accept       json
// @Produce      json
// @Param        status  query     string  true  "视频状态：pending, processing, completed, failed"
// @Success      200     {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"videos\": [...], \"count\": 1, \"status\": \"pending\"}}"
// @Failure      400     {object}  ErrorResponse  "请求参数错误（如 status 参数无效）"
// @Failure      500     {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/videos [get]
func (h *Handler) GetVideosByStatus(c *gin.Context) {
	var req GetVideosByStatusRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid status parameter",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 将字符串转换为枚举类型
	status := novel.VideoStatus(req.Status)

	// 调用Service层
	videos, err := h.novelService.GetVideosByStatus(ctx, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": GetVideosByStatusResponseData{
			Videos: toVideoInfoList(videos),
			Count:  len(videos),
			Status: req.Status,
		},
	})
}
