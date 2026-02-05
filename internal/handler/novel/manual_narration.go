package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ManualNarrationRequest 人工提交解说 JSON，生成新版本
type ManualNarrationRequest struct {
	UserID        string `json:"user_id" binding:"required"`        // 用户ID（必填）
	Prompt        string `json:"prompt,omitempty"`                  // 生成提示词（可选，便于记录）
	NarrationText string `json:"narration_text" binding:"required"` // 解说 JSON 文本（必填）
}

// ManualNarrationResponseData 人工提交解说响应
type ManualNarrationResponseData struct {
	ChapterID   string `json:"chapter_id"`
	NarrationID string `json:"narration_id"`
	Version     int    `json:"version"`
}

// CreateNarrationVersionManual 人工提交解说 JSON，生成新的解说版本
// @Summary      人工提交解说并生成新版本
// @Description  人工提交解说 JSON 文本，服务端解析/校验并生成新的解说版本（写入 narrations/scenes/shots）
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        chapter_id  path      string                true  "章节ID"
// @Param        request     body      ManualNarrationRequest true  "请求体"
// @Success      200         {object}  map[string]interface{} "成功响应"
// @Failure      400         {object}  ErrorResponse          "请求参数错误"
// @Failure      500         {object}  ErrorResponse          "服务器内部错误"
// @Router       /api/v1/novels/chapters/{chapter_id}/narration/manual [post]
func (h *Handler) CreateNarrationVersionManual(c *gin.Context) {
	chapterID := c.Param("chapter_id")
	if chapterID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "chapter_id is required",
		})
		return
	}

	var req ManualNarrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40002,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	n, err := h.novelService.CreateNarrationVersionFromText(ctx, chapterID, req.UserID, req.Prompt, req.NarrationText)
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
		"data": ManualNarrationResponseData{
			ChapterID:   chapterID,
			NarrationID: n.ID,
			Version:     n.Version,
		},
	})
}


