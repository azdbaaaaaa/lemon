package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetNovelRequest 获取小说请求
type GetNovelRequest struct {
	NovelID string `uri:"novel_id" binding:"required"` // 小说ID（必填）
}

// GetNovelResponseData 获取小说响应数据
type GetNovelResponseData struct {
	Novel NovelInfo `json:"novel"` // 小说信息
}

// GetNovel 获取小说信息
// @Summary      获取小说信息
// @Description  根据小说ID获取小说的详细信息
// @Tags         小说管理
// @Accept       json
// @Produce      json
// @Param        novel_id  path      string  true  "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"success\", \"data\": {\"novel\": {...}}}"
// @Failure      400       {object}  ErrorResponse  "请求参数错误"
// @Failure      404       {object}  ErrorResponse  "小说不存在"
// @Failure      500       {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels/{novel_id} [get]
func (h *Handler) GetNovel(c *gin.Context) {
	var req GetNovelRequest
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
	novelEntity, err := h.novelService.GetNovel(ctx, req.NovelID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "novel not found" {
			code = http.StatusNotFound
			errorCode = 40401
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": GetNovelResponseData{
			Novel: toNovelInfo(novelEntity),
		},
	})
}
