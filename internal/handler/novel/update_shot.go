package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UpdateShotRequest 更新分镜头请求
type UpdateShotRequest struct {
	Narration      *string  `json:"narration,omitempty"`       // 解说内容
	ImagePrompt    *string  `json:"image_prompt,omitempty"`    // 图片提示词
	VideoPrompt    *string  `json:"video_prompt,omitempty"`    // 视频提示词
	CameraMovement *string  `json:"camera_movement,omitempty"` // 运镜方式
	Duration       *float64 `json:"duration,omitempty"`        // 时长（秒）
}

// UpdateShot 更新分镜头信息
// @Summary      更新分镜头信息
// @Description  更新分镜头的脚本信息（解说、图片提示词、视频提示词、运镜方式、时长等）
// @Tags         分镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  path      string            true  "分镜头ID"
// @Param        request  body      UpdateShotRequest  true  "请求体"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse          "请求参数错误"
// @Failure      500      {object}  ErrorResponse          "服务器内部错误"
// @Router       /api/v1/shots/{shot_id} [put]
func (h *Handler) UpdateShot(c *gin.Context) {
	shotID := c.Param("shot_id")
	if shotID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "shot_id is required",
		})
		return
	}

	var req UpdateShotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40002,
			Message: err.Error(),
		})
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Narration != nil {
		updates["narration"] = *req.Narration
	}
	if req.ImagePrompt != nil {
		updates["image_prompt"] = *req.ImagePrompt
	}
	if req.VideoPrompt != nil {
		updates["video_prompt"] = *req.VideoPrompt
	}
	if req.CameraMovement != nil {
		updates["camera_movement"] = *req.CameraMovement
	}
	if req.Duration != nil {
		updates["duration"] = *req.Duration
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40003,
			Message: "至少需要提供一个更新字段",
		})
		return
	}

	ctx := c.Request.Context()
	if err := h.novelService.UpdateShot(ctx, shotID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"shot_id": shotID,
		},
	})
}

// RegenerateShotScript 重新生成分镜头脚本
// @Summary      重新生成分镜头脚本
// @Description  调用 LLM 重新生成分镜头的脚本信息（解说、图片提示词、视频提示词、运镜方式、时长等）
// @Tags         分镜头管理
// @Accept       json
// @Produce      json
// @Param        shot_id  path      string  true  "分镜头ID"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse          "请求参数错误"
// @Failure      500      {object}  ErrorResponse          "服务器内部错误"
// @Router       /api/v1/shots/{shot_id}/regenerate [post]
func (h *Handler) RegenerateShotScript(c *gin.Context) {
	shotID := c.Param("shot_id")
	if shotID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "shot_id is required",
		})
		return
	}

	ctx := c.Request.Context()
	if err := h.novelService.RegenerateShotScript(ctx, shotID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"shot_id": shotID,
		},
	})
}

