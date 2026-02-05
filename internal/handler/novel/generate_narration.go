package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateNarrationRequest 生成解说请求
type GenerateNarrationRequest struct {
	ChapterID string `json:"chapter_id" uri:"chapter_id" binding:"required"` // 章节ID（必填）
}

// GenerateNarrationResponseData 生成解说响应数据
type GenerateNarrationResponseData struct {
	NarrationText string `json:"narration_text"` // 生成的解说文本（JSON格式）
	NarrationID   string `json:"narration_id"`   // 解说ID（用于后续生成音频/字幕/图片/视频）
	Version       int    `json:"version"`        // 解说版本号
	ChapterID     string `json:"chapter_id"`      // 章节ID
}

// GenerateNarration 为单一章节生成解说文本
// @Summary      生成章节解说
// @Description  为单一章节生成解说文本，返回JSON格式的解说内容。解说内容会保存到数据库，包括场景和镜头信息。
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"解说生成成功\", \"data\": {\"narration_text\": \"...\", \"chapter_id\": \"...\"}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/narration [post]
func (h *Handler) GenerateNarration(c *gin.Context) {
	var req GenerateNarrationRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid chapter_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	narrationEntity, narrationText, err := h.novelService.GenerateNarrationForChapterWithMeta(ctx, req.ChapterID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "generated narrationText is empty":
			code = http.StatusBadRequest
			errorCode = 40002
		case err.Error() == "narration validation failed: 缺少 scenes 字段或 scenes 为空":
			code = http.StatusBadRequest
			errorCode = 40003
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "解说生成成功",
		"data": GenerateNarrationResponseData{
			NarrationText: narrationText,
			NarrationID:   narrationEntity.ID,
			Version:       narrationEntity.Version,
			ChapterID:     req.ChapterID,
		},
	})
}

// GenerateNarrationsForAllChaptersRequest 为所有章节生成解说请求
type GenerateNarrationsForAllChaptersRequest struct {
	NovelID string `json:"novel_id" uri:"novel_id" binding:"required"` // 小说ID（必填）
}

// GenerateNarrationsForAllChaptersResponseData 为所有章节生成解说响应数据
type GenerateNarrationsForAllChaptersResponseData struct {
	NovelID string `json:"novel_id"` // 小说ID
	Message string `json:"message"`  // 处理结果消息
}

// GenerateNarrationsForAllChapters 并发地为所有章节生成解说文本
// @Summary      为所有章节生成解说
// @Description  并发地为小说的所有章节生成解说文本。这是一个异步操作，会为每个章节生成解说并保存到数据库。
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        novel_id  path      string  true  "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"所有章节解说生成任务已提交\", \"data\": {\"novel_id\": \"...\", \"message\": \"...\"}}"
// @Failure      400       {object}  ErrorResponse  "请求参数错误"
// @Failure      500       {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/{novel_id}/chapters/narration [post]
func (h *Handler) GenerateNarrationsForAllChapters(c *gin.Context) {
	var req GenerateNarrationsForAllChaptersRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid novel_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	err := h.novelService.GenerateNarrationsForAllChapters(ctx, req.NovelID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "no chapters found":
			code = http.StatusBadRequest
			errorCode = 40002
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "所有章节解说生成任务已提交",
		"data": GenerateNarrationsForAllChaptersResponseData{
			NovelID: req.NovelID,
			Message: "所有章节解说生成任务已提交",
		},
	})
}
