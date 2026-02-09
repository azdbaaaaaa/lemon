package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateFinalVideoRequest 生成最终视频请求
type GenerateFinalVideoRequest struct {
	ChapterID string `json:"chapter_id" binding:"required"` // 章节ID（必填）
	Version   int    `json:"version,omitempty"`             // 可选：指定要合并的视频版本号
}

// GenerateFinalVideoResponseData 生成最终视频响应数据
type GenerateFinalVideoResponseData struct {
	VideoID   string `json:"video_id"`   // 生成的最终视频ID
	ChapterID string `json:"chapter_id"` // 章节ID
}

// GenerateFinalVideo 生成章节的最终完整视频
// @Summary      生成章节的最终完整视频
// @Description  拼接所有 shot 视频，添加 finish.mp4，生成章节的最终完整视频。需要确保所有 shot 视频已完成（status=completed）。
// @Tags         视频生成
// @Accept       json
// @Produce      json
// @Param        chapter_id  body      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"最终视频生成成功\", \"data\": {\"video_id\": \"...\", \"chapter_id\": \"...\"}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误（如没有找到 shot 视频）"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/chapters/videos/final [post]
func (h *Handler) GenerateFinalVideo(c *gin.Context) {
	var req GenerateFinalVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid chapter_id",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 获取版本号（如果未在 body 中提供，默认为 0）
	version := req.Version

	// 调用Service层
	videoID, err := h.novelService.GenerateFinalVideoForChapterWithVersion(ctx, req.ChapterID, version)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "no shot videos found for chapter":
			code = http.StatusBadRequest
			errorCode = 40002
		case err.Error() == "no videos found for chapter":
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
		"message": "最终视频生成成功",
		"data": GenerateFinalVideoResponseData{
			VideoID:   videoID,
			ChapterID: req.ChapterID,
		},
	})
}
