package novel

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GenerateShotVideosRequest 生成 shot 视频请求
type GenerateShotVideosRequest struct {
	ChapterID string `json:"chapter_id" binding:"required"` // 章节ID（必填）
}

// GenerateShotVideosResponseData 生成 shot 视频响应数据
type GenerateShotVideosResponseData struct {
	VideoIDs  []string `json:"video_ids"`  // 生成的视频ID列表
	Count     int      `json:"count"`      // 生成的视频数量
	ChapterID string   `json:"chapter_id"` // 章节ID
}

// GenerateShotVideos 为章节生成所有 shot 视频
// @Summary      生成章节的 shot 视频
// @Description  为章节生成所有 shot 视频，所有分镜都单独生成视频，使用图生视频方式（Ark API 或 FFmpeg）。视频生成是异步的，提交任务后需要通过状态查询接口轮询进度。
// @Tags         视频生成
// @Accept       json
// @Produce      json
// @Param        chapter_id  body      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"视频生成任务已提交\", \"data\": {\"video_ids\": [\"...\"], \"count\": 1, \"chapter_id\": \"...\"}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/chapters/videos/shots [post]
func (h *Handler) GenerateShotVideos(c *gin.Context) {
	var req GenerateShotVideosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid chapter_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	videoIDs, err := h.novelService.GenerateVideosForChapter(ctx, req.ChapterID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "chapter has no active scene version":
			code = http.StatusBadRequest
			errorCode = 40002
		case strings.Contains(err.Error(), "no shots found"):
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
		"data": GenerateShotVideosResponseData{
			VideoIDs:  videoIDs,
			Count:     len(videoIDs),
			ChapterID: req.ChapterID,
		},
	})
}

// GenerateVideosForChapterRequest 为章节生成视频请求
type GenerateVideosForChapterRequest struct {
	ChapterID string `json:"chapter_id" binding:"required"` // 章节ID（必填）
}

// GenerateVideosForChapterResponseData 为章节生成视频响应数据
type GenerateVideosForChapterResponseData struct {
	VideoIDs  []string `json:"video_ids"`  // 生成的视频ID列表
	Count     int      `json:"count"`      // 生成的视频数量
	ChapterID string   `json:"chapter_id"` // 章节ID
	Message   string   `json:"message"`    // 提示信息
}

// GenerateVideosForChapter 为章节的所有 shot 生成视频
// @Summary      生成章节视频
// @Description  为章节的所有 shot 生成视频，基于章节的 active_scene_version 获取所有 shot，为每个 shot 生成视频。视频生成是异步的，提交任务后需要通过状态查询接口轮询进度。
// @Tags         视频生成
// @Accept       json
// @Produce      json
// @Param        chapter_id  body      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/chapters/videos [post]
func (h *Handler) GenerateVideosForChapter(c *gin.Context) {
	var req GenerateVideosForChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid chapter_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	videoIDs, err := h.novelService.GenerateVideosForChapter(ctx, req.ChapterID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "chapter has no active scene version":
			code = http.StatusBadRequest
			errorCode = 40002
		case strings.Contains(err.Error(), "no shots found"):
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
		"data": GenerateVideosForChapterResponseData{
			VideoIDs:  videoIDs,
			Count:     len(videoIDs),
			ChapterID: req.ChapterID,
			Message:   "视频生成任务已启动，请通过状态查询接口获取进度",
		},
	})
}
