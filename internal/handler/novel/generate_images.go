package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateImagesRequest 生成图片请求
type GenerateImagesRequest struct {
	NarrationID string `json:"narration_id" uri:"narration_id" binding:"required"` // 解说ID（必填）
}

// GenerateImagesResponseData 生成图片响应数据
type GenerateImagesResponseData struct {
	ImageIDs    []string `json:"image_ids"`    // 生成的图片ID列表
	Count       int      `json:"count"`         // 生成的图片数量
	NarrationID string   `json:"narration_id"` // 解说ID
}

// GenerateImages 为章节解说生成所有章节图片
// @Summary      生成章节图片
// @Description  为章节解说生成所有章节图片，使用图片生成服务（Ark API）生成图片。图片生成是异步的，提交任务后需要通过状态查询接口轮询进度。
// @Tags         图片生成
// @Accept       json
// @Produce      json
// @Param        narration_id  path      string  true  "解说ID"
// @Success      200           {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"图片生成任务已提交\", \"data\": {\"image_ids\": [\"...\"], \"count\": 1, \"narration_id\": \"...\"}}"
// @Failure      400           {object}  ErrorResponse  "请求参数错误"
// @Failure      500           {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/narrations/{narration_id}/images [post]
func (h *Handler) GenerateImages(c *gin.Context) {
	var req GenerateImagesRequest
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
	imageIDs, err := h.novelService.GenerateImagesForNarration(ctx, req.NarrationID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "find narration":
			code = http.StatusNotFound
			errorCode = 40401
		case err.Error() == "no scenes found for narration":
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
		"message": "图片生成任务已提交",
		"data": GenerateImagesResponseData{
			ImageIDs:    imageIDs,
			Count:       len(imageIDs),
			NarrationID: req.NarrationID,
		},
	})
}

// GetImageVersionsResponseData 获取图片版本列表响应数据
type GetImageVersionsResponseData struct {
	ChapterID string `json:"chapter_id"` // 章节ID
	Versions   []int     `json:"versions"` // 版本号列表
}

// GetImageVersions 获取章节的所有图片版本号
// @Summary      获取图片版本列表
// @Description  获取章节的所有图片版本号列表。
// @Tags         图片生成
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"获取成功\", \"data\": {\"chapter_id\": \"...\", \"versions\": [1, 2, 3]}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/images/versions [get]
func (h *Handler) GetImageVersions(c *gin.Context) {
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
	versions, err := h.novelService.GetImageVersions(ctx, chapterID)
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
		"data": GetImageVersionsResponseData{
			ChapterID: chapterID,
			Versions:  versions,
		},
	})
}
