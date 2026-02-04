package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateNarrationVideosRequest 生成 narration 视频请求
type GenerateNarrationVideosRequest struct {
	ChapterID string `json:"chapter_id" uri:"chapter_id" binding:"required"` // 章节ID（必填）
}

// GenerateNarrationVideosResponseData 生成 narration 视频响应数据
type GenerateNarrationVideosResponseData struct {
	VideoIDs  []string `json:"video_ids"`  // 生成的视频ID列表
	Count     int      `json:"count"`      // 生成的视频数量
	ChapterID string   `json:"chapter_id"` // 章节ID
}

// GenerateNarrationVideos 为章节生成所有 narration 视频
// @Summary      生成章节的 narration 视频
// @Description  为章节生成所有 narration 视频，所有分镜都单独生成视频，使用图生视频方式（Ark API 或 FFmpeg）。视频生成是异步的，提交任务后需要通过状态查询接口轮询进度。
// @Tags         视频生成
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"视频生成任务已提交\", \"data\": {\"video_ids\": [\"...\"], \"count\": 1, \"chapter_id\": \"...\"}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/videos/narration [post]
func (h *Handler) GenerateNarrationVideos(c *gin.Context) {
	var req GenerateNarrationVideosRequest
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
	videoIDs, err := h.novelService.GenerateNarrationVideosForChapter(ctx, req.ChapterID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "narration content is empty":
			code = http.StatusBadRequest
			errorCode = 40002
		case err.Error() == "no shots found in narration content":
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
		"message": "视频生成任务已提交",
		"data": GenerateNarrationVideosResponseData{
			VideoIDs:  videoIDs,
			Count:     len(videoIDs),
			ChapterID: req.ChapterID,
		},
	})
}
