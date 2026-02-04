package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SplitChaptersRequest 切分章节请求
type SplitChaptersRequest struct {
	NovelID       string `json:"novel_id" binding:"required"`        // 小说ID（必填）
	TargetChapters int   `json:"target_chapters" binding:"required,min=1"` // 目标章节数（必填，至少1章）
}

// SplitChaptersResponseData 切分章节响应数据
type SplitChaptersResponseData struct {
	NovelID       string `json:"novel_id"`        // 小说ID
	TargetChapters int   `json:"target_chapters"` // 目标章节数
	Message       string `json:"message"`         // 响应消息
}

// SplitChapters 根据小说内容切分章节
// @Summary      切分章节
// @Description  根据小说内容切分章节，将小说文本按照目标章节数切分成多个章节。这是小说处理流程的第二步。
// @Tags         章节管理
// @Accept       json
// @Produce      json
// @Param        request  body      SplitChaptersRequest  true  "切分章节请求"
// @Success      200      {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"章节切分成功\", \"data\": {\"novel_id\": \"...\", \"target_chapters\": 10, \"message\": \"已切分为 10 个章节\"}}"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/{novel_id}/chapters/split [post]
func (h *Handler) SplitChapters(c *gin.Context) {
	var req SplitChaptersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	err := h.novelService.SplitNovelIntoChapters(ctx, req.NovelID, req.TargetChapters)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		switch {
		case err.Error() == "failed to find novel":
			code = http.StatusBadRequest
			errorCode = 40002
		case err.Error() == "no chapters split from novel content":
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
		"message": "章节切分成功",
		"data": SplitChaptersResponseData{
			NovelID:        req.NovelID,
			TargetChapters: req.TargetChapters,
			Message:        "已切分为章节",
		},
	})
}
