package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GeneratePropImages 为小说的所有道具生成图片
// @Summary      生成道具图片
// @Description  为小说的所有道具生成图片（抽卡）
// @Tags         图片生成
// @Accept       json
// @Produce      json
// @Param        novel_id  path      string  true  "小说ID"
// @Success      200       {object}  map[string]interface{}  "成功响应"
// @Failure      400       {object}  ErrorResponse          "请求参数错误"
// @Failure      500       {object}  ErrorResponse          "服务器内部错误"
// @Router       /api/v1/novels/{novel_id}/props/images [post]
func (h *Handler) GeneratePropImages(c *gin.Context) {
	novelID := c.Param("novel_id")
	if novelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "novel_id is required",
		})
		return
	}

	ctx := c.Request.Context()
	imageIDs, err := h.novelService.GeneratePropImages(ctx, novelID)
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
		"data": gin.H{
			"novel_id":  novelID,
			"image_ids": imageIDs,
			"count":     len(imageIDs),
		},
	})
}

