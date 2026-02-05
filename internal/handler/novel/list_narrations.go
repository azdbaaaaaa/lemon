package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListNarrationsResponseData 列出章节解说版本响应
type ListNarrationsResponseData struct {
	ChapterID  string         `json:"chapter_id"`
	Narrations []NarrationInfo `json:"narrations"`
	Count      int            `json:"count"`
}

// ListNarrationsByChapterID 列出章节的所有解说版本
// @Summary      列出章节解说版本
// @Description  列出指定章节的所有解说版本（包含 narration_id、version、prompt、status）
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string  true  "章节ID"
// @Success      200         {object}  map[string]interface{}  "成功响应"
// @Failure      400         {object}  ErrorResponse  "请求参数错误"
// @Failure      500         {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/narrations [get]
func (h *Handler) ListNarrationsByChapterID(c *gin.Context) {
	chapterID := c.Param("chapter_id")
	if chapterID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "chapter_id is required",
		})
		return
	}

	ctx := c.Request.Context()
	narrations, err := h.novelService.ListNarrationsByChapterID(ctx, chapterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	infos := make([]NarrationInfo, 0, len(narrations))
	for _, n := range narrations {
		infos = append(infos, toNarrationInfo(n))
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": ListNarrationsResponseData{
			ChapterID:  chapterID,
			Narrations: infos,
			Count:      len(infos),
		},
	})
}


