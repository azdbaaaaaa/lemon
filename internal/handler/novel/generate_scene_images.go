package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GenerateSceneImages 为解说的所有场景生成图片
// @Summary      生成场景图片
// @Description  为解说的所有场景生成图片（抽卡）
// @Tags         图片生成
// @Accept       json
// @Produce      json
// @Param        narration_id  path      string  true  "解说ID"
// @Success      200           {object}  map[string]interface{}  "成功响应"
// @Failure      400           {object}  ErrorResponse          "请求参数错误"
// @Failure      500           {object}  ErrorResponse          "服务器内部错误"
// @Router       /api/v1/narrations/{narration_id}/scenes/images [post]
func (h *Handler) GenerateSceneImages(c *gin.Context) {
	narrationID := c.Param("narration_id")
	if narrationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "narration_id is required",
		})
		return
	}

	ctx := c.Request.Context()
	imageIDs, err := h.novelService.GenerateSceneImages(ctx, narrationID)
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
			"narration_id": narrationID,
			"image_ids":    imageIDs,
			"count":        len(imageIDs),
		},
	})
}

