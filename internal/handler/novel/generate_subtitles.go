package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateSubtitlesRequest 生成字幕请求
type GenerateSubtitlesRequest struct {
	NarrationID string `json:"narration_id" uri:"narration_id" binding:"required"` // 解说ID（必填）
}

// GenerateSubtitlesResponseData 生成字幕响应数据
type GenerateSubtitlesResponseData struct {
	SubtitleIDs []string `json:"subtitle_ids"` // 生成的字幕ID列表
	Count       int      `json:"count"`        // 生成的字幕数量
	NarrationID string   `json:"narration_id"` // 解说ID
}

// GenerateSubtitles 为章节解说生成所有字幕文件（ASS格式）
// @Summary      生成章节字幕
// @Description  为章节解说生成所有字幕文件（ASS格式），为每个 narration shot 生成单独的字幕文件，与音频片段一一对应。需要先有章节音频记录（包含时间戳数据）。
// @Tags         字幕生成
// @Accept       json
// @Produce      json
// @Param        narration_id  path      string  true  "解说ID"
// @Success      200           {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"字幕生成任务已提交\", \"data\": {\"subtitle_ids\": [\"...\"], \"count\": 1, \"narration_id\": \"...\"}}"
// @Failure      400           {object}  ErrorResponse  "请求参数错误"
// @Failure      500           {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/narrations/{narration_id}/subtitles [post]
func (h *Handler) GenerateSubtitles(c *gin.Context) {
	var req GenerateSubtitlesRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid narration_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	subtitleIDs, err := h.novelService.GenerateSubtitlesForNarration(ctx, req.NarrationID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "failed to find narration":
			code = http.StatusNotFound
			errorCode = 40401
		case err.Error() == "no shots found in narration":
			code = http.StatusBadRequest
			errorCode = 40002
		case err.Error() == "no audio found":
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
		"message": "字幕生成任务已提交",
		"data": GenerateSubtitlesResponseData{
			SubtitleIDs: subtitleIDs,
			Count:       len(subtitleIDs),
			NarrationID: req.NarrationID,
		},
	})
}

// GetSubtitleVersionsResponseData 获取字幕版本列表响应数据
type GetSubtitleVersionsResponseData struct {
	ChapterID string `json:"chapter_id"` // 章节ID
	Versions  []int  `json:"versions"`   // 版本号列表
}

// GetSubtitleVersions 获取章节的所有字幕版本号
// @Summary      获取字幕版本列表
// @Description  获取章节的所有字幕版本号列表。
// @Tags         字幕生成
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": {\"chapter_id\": \"...\", \"versions\": [1, 2, 3]}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/subtitles/versions [get]
func (h *Handler) GetSubtitleVersions(c *gin.Context) {
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
	versions, err := h.novelService.GetSubtitleVersions(ctx, chapterID)
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
		"data": GetSubtitleVersionsResponseData{
			ChapterID: chapterID,
			Versions:  versions,
		},
	})
}
