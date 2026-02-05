package novel

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"lemon/internal/model/novel"
)

// SceneInfo 场景信息 DTO
type SceneInfo struct {
	ID          string `json:"id"`
	NarrationID string `json:"narration_id"`
	ChapterID   string `json:"chapter_id"`
	UserID      string `json:"user_id"`
	SceneNumber string `json:"scene_number"`
	Narration   string `json:"narration,omitempty"`
	Sequence    int    `json:"sequence"`
	Version     int    `json:"version"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toSceneInfo(s *novel.Scene) SceneInfo {
	return SceneInfo{
		ID:          s.ID,
		NarrationID: s.NarrationID,
		ChapterID:   s.ChapterID,
		UserID:      s.UserID,
		SceneNumber: s.SceneNumber,
		Narration:   s.Narration,
		Sequence:    s.Sequence,
		Version:     s.Version,
		Status:      string(s.Status),
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   s.UpdatedAt.Format(time.RFC3339),
	}
}

// GetScenesByNarration 获取解说的场景列表
// @Summary      获取解说场景列表
// @Description  根据 narration_id 获取场景列表（按 sequence 排序）
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        narration_id  path      string  true  "解说ID"
// @Success      200           {object}  map[string]interface{}  "成功响应"
// @Failure      400           {object}  ErrorResponse          "请求参数错误"
// @Failure      500           {object}  ErrorResponse          "服务器内部错误"
// @Router       /api/v1/narrations/{narration_id}/scenes [get]
func (h *Handler) GetScenesByNarration(c *gin.Context) {
	narrationID := c.Param("narration_id")
	if narrationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "narration_id is required",
		})
		return
	}

	ctx := c.Request.Context()
	scenes, err := h.novelService.GetScenesByNarrationID(ctx, narrationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	infos := make([]SceneInfo, 0, len(scenes))
	for _, s := range scenes {
		infos = append(infos, toSceneInfo(s))
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"narration_id": narrationID,
			"scenes":       infos,
			"count":        len(infos),
		},
	})
}


