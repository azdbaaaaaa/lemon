package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateAudiosRequest 生成音频请求
type GenerateAudiosRequest struct {
	NarrationID string `json:"narration_id" uri:"narration_id" binding:"required"` // 解说ID（必填）
}

// GenerateAudiosResponseData 生成音频响应数据
type GenerateAudiosResponseData struct {
	AudioIDs    []string `json:"audio_ids"`    // 生成的音频ID列表
	Count       int      `json:"count"`         // 生成的音频数量
	NarrationID string   `json:"narration_id"`  // 解说ID
}

// GenerateAudios 为章节解说生成所有章节音频片段
// @Summary      生成章节音频
// @Description  为章节解说生成所有章节音频片段，使用TTS服务生成音频。音频生成是异步的，提交任务后需要通过状态查询接口轮询进度。
// @Tags         音频生成
// @Accept       json
// @Produce      json
// @Param        narration_id  path      string  true  "解说ID"
// @Success      200           {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"音频生成任务已提交\", \"data\": {\"audio_ids\": [\"...\"], \"count\": 1, \"narration_id\": \"...\"}}"
// @Failure      400           {object}  ErrorResponse  "请求参数错误"
// @Failure      500           {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/narrations/{narration_id}/audios [post]
func (h *Handler) GenerateAudios(c *gin.Context) {
	var req GenerateAudiosRequest
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
	audioIDs, err := h.novelService.GenerateAudiosForNarration(ctx, req.NarrationID)
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
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "音频生成任务已提交",
		"data": GenerateAudiosResponseData{
			AudioIDs:    audioIDs,
			Count:       len(audioIDs),
			NarrationID: req.NarrationID,
		},
	})
}

// GetAudioVersionsResponseData 获取音频版本列表响应数据
type GetAudioVersionsResponseData struct {
	NarrationID string `json:"narration_id"` // 解说ID
	Versions    []int  `json:"versions"`      // 版本号列表
}

// GetAudioVersions 获取章节解说的所有音频版本号
// @Summary      获取音频版本列表
// @Description  获取章节解说的所有音频版本号列表。
// @Tags         音频生成
// @Accept       json
// @Produce      json
// @Param        narration_id  path      string  true  "解说ID"
// @Success      200           {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": {\"narration_id\": \"...\", \"versions\": [1, 2, 3]}}"
// @Failure      400           {object}  ErrorResponse  "请求参数错误"
// @Failure      500           {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/narrations/{narration_id}/audios/versions [get]
func (h *Handler) GetAudioVersions(c *gin.Context) {
	narrationID := c.Param("narration_id")
	if narrationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "narration_id is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	versions, err := h.novelService.GetAudioVersions(ctx, narrationID)
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
		"data": GetAudioVersionsResponseData{
			NarrationID: narrationID,
			Versions:    versions,
		},
	})
}
