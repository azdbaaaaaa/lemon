package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetChaptersRequest 获取章节列表请求
type GetChaptersRequest struct {
	NovelID string `uri:"novel_id" binding:"required"` // 小说ID（必填）
}

// GetChaptersResponseData 获取章节列表响应数据
type GetChaptersResponseData struct {
	NovelID  string       `json:"novel_id"`  // 小说ID
	Chapters []ChapterInfo `json:"chapters"`  // 章节列表
	Count    int          `json:"count"`     // 章节数量
}

// GetChapters 获取小说的所有章节
// @Summary      获取章节列表
// @Description  根据小说ID获取该小说的所有章节列表，按序号排序
// @Tags         章节管理
// @Accept       json
// @Produce      json
// @Param        novel_id  path      string  true  "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"novel_id\": \"...\", \"chapters\": [...], \"count\": 10}}"
// @Failure      400       {object}  ErrorResponse  "请求参数错误"
// @Failure      500       {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/{novel_id}/chapters [get]
func (h *Handler) GetChapters(c *gin.Context) {
	var req GetChaptersRequest
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
	chapters, err := h.novelService.GetChapters(ctx, req.NovelID)
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
		"data": GetChaptersResponseData{
			NovelID:  req.NovelID,
			Chapters: toChapterInfoList(chapters),
			Count:    len(chapters),
		},
	})
}
