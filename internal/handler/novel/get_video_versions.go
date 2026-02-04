package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetVideoVersionsRequest 获取视频版本请求
type GetVideoVersionsRequest struct {
	ChapterID string `json:"chapter_id" uri:"chapter_id" binding:"required"` // 章节ID（必填）
}

// GetVideoVersionsResponseData 获取视频版本响应数据
type GetVideoVersionsResponseData struct {
	Versions  []int  `json:"versions"`   // 视频版本号列表
	Count     int    `json:"count"`      // 版本数量
	ChapterID string `json:"chapter_id"` // 章节ID
}

// GetVideoVersions 获取章节的所有视频版本号
// @Summary      获取章节的视频版本号列表
// @Description  获取章节的所有视频版本号，用于查看历史版本或选择特定版本。每次生成视频都会创建新版本。
// @Tags         视频查询
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"versions\": [1, 2, 3], \"count\": 3, \"chapter_id\": \"...\"}}"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/videos/versions [get]
func (h *Handler) GetVideoVersions(c *gin.Context) {
	var req GetVideoVersionsRequest
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
	versions, err := h.novelService.GetVideoVersions(ctx, req.ChapterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": GetVideoVersionsResponseData{
			Versions:  versions,
			Count:     len(versions),
			ChapterID: req.ChapterID,
		},
	})
}
