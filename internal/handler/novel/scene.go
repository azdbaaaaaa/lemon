package novel

import (
	"net/http"
	"time"

	httputil "lemon/internal/pkg/http"

	"lemon/internal/model/novel"

	"github.com/gin-gonic/gin"
)

// SceneInfo 场景信息 DTO
type SceneInfo struct {
	ID          string `json:"id"`
	ChapterID   string `json:"chapter_id"`
	UserID      string `json:"user_id"`
	Description string `json:"description"` // 场景描述
	Sequence    int    `json:"sequence"`    // 序号
	Version     int    `json:"version"`     // 版本号
	Status      string `json:"status"`      // 状态
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// FromSceneEntity 从 Scene 实体创建 SceneInfo
func (info *SceneInfo) FromSceneEntity(s *novel.Scene) {
	info.ID = s.ID
	info.ChapterID = s.ChapterID
	info.UserID = s.UserID
	info.Description = s.Description
	info.Sequence = s.Sequence
	info.Version = s.Version
	info.Status = string(s.Status)
	info.CreatedAt = s.CreatedAt.Format(time.RFC3339)
	info.UpdatedAt = s.UpdatedAt.Format(time.RFC3339)
}

// GetScenesRequest 获取场景列表请求
type GetScenesRequest struct {
	ChapterID string `form:"chapter_id" binding:"required"` // 章节ID（必填）
}

// GetScenes 获取场景列表
// @Summary      获取场景列表
// @Description  根据章节ID获取场景列表（按 sequence 排序）
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        chapter_id  query     string  true   "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"
// @Failure      400         {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500         {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/scenes [get]
func (h *Handler) GetScenes(c *gin.Context) {
	var req GetScenesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request parameters",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	scenes, err := h.novelService.GetScenesByChapterID(ctx, req.ChapterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	infos := make([]SceneInfo, 0, len(scenes))
	for _, s := range scenes {
		var info SceneInfo
		info.FromSceneEntity(s)
		infos = append(infos, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"chapter_id": req.ChapterID,
			"scenes":     infos,
			"count":      len(infos),
		},
	})
}

// GenerateScenesRequest 生成场景和镜头请求
type GenerateScenesRequest struct {
	ChapterID string `json:"chapter_id" binding:"required"` // 章节ID（必填）
}

// GenerateScenesResponseData 生成场景和镜头响应数据
type GenerateScenesResponseData struct {
	ChapterID string `json:"chapter_id"` // 章节ID
	Message   string `json:"message"`    // 响应消息
}

// GenerateScenes 为章节生成场景和镜头
// @Summary      生成场景和镜头
// @Description  为指定章节生成场景和镜头（10个场景，每个场景1-4个镜头），同时会生成和保存角色、道具信息
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      GenerateScenesRequest  true  "生成场景和镜头请求"
// @Success      200     {object}  map[string]interface{}  "成功响应"
// @Failure      400     {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500     {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/scenes/generate [post]
func (h *Handler) GenerateScenes(c *gin.Context) {
	var req GenerateScenesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 通过章节ID获取章节信息
	chapter, err := h.novelService.GetChapterByID(ctx, req.ChapterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40002,
			Message: "Chapter not found",
			Detail:  err.Error(),
		})
		return
	}

	// 调用Service层生成场景和镜头
	err = h.novelService.CreateDefaultScenesAndShots(ctx, chapter.NovelID, req.ChapterID, chapter.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: "Failed to generate scenes and shots",
			Detail:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "场景和镜头生成成功",
		"data": GenerateScenesResponseData{
			ChapterID: req.ChapterID,
			Message:   "场景和镜头生成完成",
		},
	})
}

// SetActiveSceneVersionRequest 设置生效场景版本请求
type SetActiveSceneVersionRequest struct {
	ChapterID string `json:"chapter_id" binding:"required"` // 章节ID（必填）
	Version   int    `json:"version" binding:"required"`    // 版本号（必填，从1开始）
}

// SetActiveSceneVersionResponseData 设置生效场景版本响应数据
type SetActiveSceneVersionResponseData struct {
	ChapterID string `json:"chapter_id"` // 章节ID
	Version   int    `json:"version"`    // 生效版本号
	Message   string `json:"message"`    // 响应消息
}

// SetActiveSceneVersion 设置章节的生效场景版本号
// @Summary      设置生效场景版本
// @Description  设置指定章节的生效场景版本号，查询时会返回该版本的场景和镜头
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      SetActiveSceneVersionRequest  true  "设置生效场景版本请求"
// @Success      200     {object}  map[string]interface{}  "成功响应"
// @Failure      400     {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500     {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/scenes/version [put]
func (h *Handler) SetActiveSceneVersion(c *gin.Context) {
	var req SetActiveSceneVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	if req.Version <= 0 {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40002,
			Message: "Version must be greater than 0",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层设置生效版本
	err := h.novelService.SetActiveSceneVersion(ctx, req.ChapterID, req.Version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: "Failed to set active scene version",
			Detail:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "生效场景版本设置成功",
		"data": SetActiveSceneVersionResponseData{
			ChapterID: req.ChapterID,
			Version:   req.Version,
			Message:   "生效场景版本已更新",
		},
	})
}

// GetShotsRequest 获取镜头列表请求
type GetShotsRequest struct {
	ChapterID string `form:"chapter_id"` // 章节ID（获取列表时必填）
	ShotID    string `form:"shot_id"`    // 镜头ID（获取单个镜头详情时必填）
	Version   int    `form:"version"`     // 版本号（可选，不传则使用最新版本）
}

// ShotInfo 镜头信息 DTO
type ShotInfo struct {
	ID               string  `json:"id"`
	SceneID          string  `json:"scene_id"`
	ChapterID        string  `json:"chapter_id"`
	UserID           string  `json:"user_id"`
	Narration        string  `json:"narration"`
	Duration         float64 `json:"duration,omitempty"`
	FirstImagePrompt string  `json:"first_image_prompt"`
	LastImagePrompt  string  `json:"last_image_prompt"`
	VideoPrompt      string  `json:"video_prompt"`
	Sequence         int     `json:"sequence"`
	Version          int     `json:"version"`
	FirstImageID     string  `json:"first_image_id,omitempty"`
	LastImageID      string  `json:"last_image_id,omitempty"`
	Status           string  `json:"status"`
	ErrorMessage     string  `json:"error_message,omitempty"`
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
	info.FirstImageID = s.FirstImageID
	info.LastImageID = s.LastImageID
	info.Status = string(s.Status)
	info.ErrorMessage = s.ErrorMessage
	info.CreatedAt = s.CreatedAt.Format(time.RFC3339)
	info.UpdatedAt = s.UpdatedAt.Format(time.RFC3339)
}

// GetShots 获取镜头列表或详情
// @Summary      获取镜头列表或详情
// @Description  根据章节ID获取镜头列表，或根据镜头ID获取单个镜头详情
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        chapter_id  query     string  false  "章节ID（获取列表时必填）"
// @Param        shot_id     query     string  false  "镜头ID（获取单个镜头详情时必填）"
// @Param        version     query     int     false  "版本号（可选）"
// @Success      200         {object}  map[string]interface{}  "成功响应"
// @Failure      400         {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500         {object}  httputil.ErrorResponse  "服务器内部错误"
// @Router       /api/v1/shots [get]
func (h *Handler) GetShots(c *gin.Context) {
	shotID := c.Query("shot_id")
	if shotID != "" {
		// 获取单个镜头详情
		ctx := c.Request.Context()
		shot, err := h.novelService.GetShotByID(ctx, shotID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Code:    50001,
				Message: err.Error(),
			})
			return
		}

		var info ShotInfo
		info.FromShotEntity(shot)

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"shot": info,
			},
		})
		return
	}

	// 获取镜头列表
	chapterID := c.Query("chapter_id")
	if chapterID == "" {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40001,
			Message: "chapter_id or shot_id is required",
		})
		return
	}

	ctx := c.Request.Context()
	shots, err := h.novelService.GetShotsByChapterID(ctx, chapterID)
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
			"chapter_id": chapterID,
			"shots":      infos,
			"count":      len(infos),
		},
	})
}

// UpdateShotRequest 更新镜头请求
type UpdateShotRequest struct {
	ShotID           string   `json:"shot_id" binding:"required"`              // 镜头ID（必填）
	Narration        *string  `json:"narration,omitempty"`                      // 旁白（可选）
	Duration         *float64 `json:"duration,omitempty"`                       // 时长（可选）
	FirstImagePrompt *string  `json:"first_image_prompt,omitempty"`              // 首图提示词（可选）
	LastImagePrompt  *string  `json:"last_image_prompt,omitempty"`               // 末图提示词（可选）
	VideoPrompt      *string  `json:"video_prompt,omitempty"`                    // 视频提示词（可选）
	FirstImageID     *string  `json:"first_image_id,omitempty"`                  // 首图ID（可选）
	LastImageID      *string  `json:"last_image_id,omitempty"`                   // 末图ID（可选）
}

// UpdateShot 更新镜头信息
// @Summary      更新镜头信息
// @Description  更新指定镜头的信息（旁白、时长、提示词等）
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        request  body      UpdateShotRequest  true  "更新镜头请求"
// @Success      200     {object}  map[string]interface{}  "成功响应"
// @Failure      400     {object}  httputil.ErrorResponse  "请求参数错误"
// @Failure      500     {object}  httputil.ErrorResponse  "服务器内部错误"
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

	ctx := c.Request.Context()

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Narration != nil {
		updates["narration"] = *req.Narration
	}
	if req.Duration != nil {
		updates["duration"] = *req.Duration
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
	if req.FirstImageID != nil {
		updates["first_image_id"] = *req.FirstImageID
	}
	if req.LastImageID != nil {
		updates["last_image_id"] = *req.LastImageID
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
			Code:    40002,
			Message: "No fields to update",
		})
		return
	}

	// 调用Service层更新镜头
	err := h.novelService.UpdateShot(ctx, req.ShotID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
			Code:    50001,
			Message: "Failed to update shot",
			Detail:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "镜头信息更新成功",
		"data": gin.H{
			"shot_id": req.ShotID,
		},
	})
}
