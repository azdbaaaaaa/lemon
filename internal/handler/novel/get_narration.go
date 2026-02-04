package novel

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"lemon/internal/model/novel"
)

// NarrationInfo 解说信息 DTO
type NarrationInfo struct {
	ID        string `json:"id"`         // 解说ID
	ChapterID string `json:"chapter_id"`  // 章节ID
	UserID    string `json:"user_id"`    // 用户ID
	Prompt    string `json:"prompt"`     // 生成提示词
	Version   int    `json:"version"`    // 版本号
	Status    string `json:"status"`     // 状态：pending, completed, failed
	CreatedAt string `json:"created_at"` // 创建时间
	UpdatedAt string `json:"updated_at"` // 更新时间
}

// toNarrationInfo 将 Narration 实体转换为 NarrationInfo DTO
func toNarrationInfo(narrationEntity *novel.Narration) NarrationInfo {
	return NarrationInfo{
		ID:        narrationEntity.ID,
		ChapterID: narrationEntity.ChapterID,
		UserID:    narrationEntity.UserID,
		Prompt:    narrationEntity.Prompt,
		Version:   narrationEntity.Version,
		Status:    string(narrationEntity.Status),
		CreatedAt: narrationEntity.CreatedAt.Format(time.RFC3339),
		UpdatedAt: narrationEntity.UpdatedAt.Format(time.RFC3339),
	}
}

// GetNarration 根据章节ID获取章节解说（返回最新版本）
// @Summary      获取章节解说
// @Description  根据章节ID获取章节解说，返回最新版本的解说信息。
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": {...}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      404         {object}  ErrorResponse  "解说不存在"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/narration [get]
func (h *Handler) GetNarration(c *gin.Context) {
	chapterID := c.Param("chapter_id")
	if chapterID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "chapter_id is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	narration, err := h.novelService.GetNarration(ctx, chapterID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "narration not found" {
			code = http.StatusNotFound
			errorCode = 40401
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    toNarrationInfo(narration),
	})
}

// GetNarrationByVersion 根据章节ID和版本号获取章节解说
// @Summary      获取指定版本的章节解说
// @Description  根据章节ID和版本号获取指定版本的章节解说信息。
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string  true  "章节ID"
// @Param        version     query     int     true  "版本号"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": {...}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      404         {object}  ErrorResponse  "解说不存在"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/narration/version/{version} [get]
func (h *Handler) GetNarrationByVersion(c *gin.Context) {
	chapterID := c.Param("chapter_id")
	if chapterID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "chapter_id is required",
		})
		return
	}

	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40002,
			Message: "Invalid version",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	narration, err := h.novelService.GetNarrationByVersion(ctx, chapterID, version)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "narration not found" {
			code = http.StatusNotFound
			errorCode = 40401
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    toNarrationInfo(narration),
	})
}

// GetNarrationVersionsResponseData 获取章节解说版本列表响应数据
type GetNarrationVersionsResponseData struct {
	ChapterID string `json:"chapter_id"` // 章节ID
	Versions  []int  `json:"versions"`   // 版本号列表
}

// GetNarrationVersions 获取章节的所有版本号
// @Summary      获取章节解说版本列表
// @Description  获取章节的所有解说版本号列表。
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": {\"chapter_id\": \"...\", \"versions\": [1, 2, 3]}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/narration/versions [get]
func (h *Handler) GetNarrationVersions(c *gin.Context) {
	chapterID := c.Param("chapter_id")
	if chapterID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "chapter_id is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	versions, err := h.novelService.GetNarrationVersions(ctx, chapterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data": GetNarrationVersionsResponseData{
			ChapterID: chapterID,
			Versions:  versions,
		},
	})
}

// SetNarrationVersionRequest 设置解说版本请求
type SetNarrationVersionRequest struct {
	NarrationID string `json:"narration_id" binding:"required"` // 解说ID（必填）
	Version     int    `json:"version" binding:"required"`      // 版本号（必填）
}

// SetNarrationVersion 设置章节解说的版本号
// @Summary      设置解说版本号
// @Description  设置章节解说的版本号。
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        request  body      SetNarrationVersionRequest  true  "设置版本请求"
// @Success      200      {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"版本号设置成功\"}"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/narrations/{narration_id}/version [put]
func (h *Handler) SetNarrationVersion(c *gin.Context) {
	var req SetNarrationVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	// 从 URI 获取 narration_id（如果提供）
	narrationID := c.Param("narration_id")
	if narrationID != "" {
		req.NarrationID = narrationID
	}

	ctx := c.Request.Context()

	// 调用Service层
	err := h.novelService.SetNarrationVersion(ctx, req.NarrationID, req.Version)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "narration not found" {
			code = http.StatusNotFound
			errorCode = 40401
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "版本号设置成功",
	})
}
