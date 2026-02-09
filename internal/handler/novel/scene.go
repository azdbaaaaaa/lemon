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
