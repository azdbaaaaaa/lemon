package novel

import (
	"net/http"
	"time"

	httputil "lemon/internal/pkg/http"

	"lemon/internal/model/novel"

	"github.com/gin-gonic/gin"
)

// ShotInfo 镜头信息 DTO
type ShotInfo struct {
	ID               string  `json:"id"`
	SceneID          string  `json:"scene_id"`
	ChapterID        string  `json:"chapter_id"`
	UserID           string  `json:"user_id"`
	Narration        string  `json:"narration"`          // 旁白（镜头解说内容）
	Duration         float64 `json:"duration,omitempty"` // 时长（秒）
	FirstImagePrompt string  `json:"first_image_prompt"` // 首图提示词（第一帧图片）
	LastImagePrompt  string  `json:"last_image_prompt"`  // 末图提示词（最后一帧图片）
	VideoPrompt      string  `json:"video_prompt"`       // 视频提示词（动态视频）
	Sequence         int     `json:"sequence"`           // 序号（在场景中的顺序）
	Version          int     `json:"version"`            // 版本号
	Status           string  `json:"status"`
	ErrorMessage     string  `json:"error_message,omitempty"` // 错误信息（失败时）
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// FromShotEntity 从 Shot 实体创建 ShotInfo
func (info *ShotInfo) FromShotEntity(s *novel.Shot) {
	info.ID = s.ID
	info.SceneID = s.SceneID
	info.ChapterID = s.ChapterID
	info.UserID = s.UserID
	info.Narration = s.Narration
	info.Duration = s.Duration
	info.FirstImagePrompt = s.FirstImagePrompt
	info.LastImagePrompt = s.LastImagePrompt
	info.VideoPrompt = s.VideoPrompt
	info.Sequence = s.Sequence
	info.Version = s.Version
	info.Status = string(s.Status)
	info.ErrorMessage = s.ErrorMessage
	info.CreatedAt = s.CreatedAt.Format(time.RFC3339)
	info.UpdatedAt = s.UpdatedAt.Format(time.RFC3339)
}

// GetShotsRequest 获取镜头列表请求
type GetShotsRequest struct {
	ChapterID string `form:"chapter_id" binding:"required"` // 章节ID（必填）
}

// GetShots 获取镜头列表
// @Summary      获取镜头列表
// @Description  根据章节ID获取镜头列表（按 index 排序）。镜头中包含解说内容。
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        chapter_id  query     string  true   "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"
// @Failure      400         {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500         {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots [get]
func (h *Handler) GetShots(c *gin.Context) {
	var req GetShotsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request parameters",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	shots, err := h.novelService.GetShotsByChapterID(ctx, req.ChapterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	infos := make([]ShotInfo, 0, len(shots))
	for _, s := range shots {
		var info ShotInfo
		info.FromShotEntity(s)
		infos = append(infos, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"chapter_id": req.ChapterID,
			"shots":      infos,
			"count":      len(infos),
		},
	})
}

// UpdateShotRequest 更新分镜头请求
type UpdateShotRequest struct {
	ShotID           string   `json:"shot_id" binding:"required"`   // 分镜头ID（必填）
	Narration        *string  `json:"narration,omitempty"`          // 解说内容
	FirstImagePrompt *string  `json:"first_image_prompt,omitempty"` // 首图提示词
	LastImagePrompt  *string  `json:"last_image_prompt,omitempty"`  // 末图提示词
	VideoPrompt      *string  `json:"video_prompt,omitempty"`       // 视频提示词
	Duration         *float64 `json:"duration,omitempty"`           // 时长（秒）
}

// UpdateShot 更新分镜头信息
// @Summary      更新分镜头信息
// @Description  更新分镜头的脚本信息（解说、图片提示词、视频提示词、运镜方式、时长等）
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      UpdateShotRequest  true  "请求体"
// @Success      200      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500      {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots [put]
func (h *Handler) UpdateShot(c *gin.Context) {
	var req UpdateShotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Narration != nil {
		updates["narration"] = *req.Narration
	}
	if req.FirstImagePrompt != nil {
		updates["first_image_prompt"] = *req.FirstImagePrompt
	}
	if req.LastImagePrompt != nil {
		updates["last_image_prompt"] = *req.LastImagePrompt
	}
	if req.VideoPrompt != nil {
		updates["video_prompt"] = *req.VideoPrompt
	}
	if req.Duration != nil {
		updates["duration"] = *req.Duration
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40002,
			Message: "至少需要提供一个更新字段",
		})
		return
	}

	ctx := c.Request.Context()
	if err := h.novelService.UpdateShot(ctx, req.ShotID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"shot_id": req.ShotID,
		},
	})
}
