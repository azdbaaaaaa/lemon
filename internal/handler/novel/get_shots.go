package novel

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"lemon/internal/model/novel"
)

// ShotInfo 镜头信息 DTO
type ShotInfo struct {
	ID          string `json:"id"`
	SceneID     string `json:"scene_id"`
	SceneNumber string `json:"scene_number"`
	NarrationID string `json:"narration_id"`
	ChapterID   string `json:"chapter_id"`
	UserID      string `json:"user_id"`

	ShotNumber     string  `json:"shot_number"`
	Character      string  `json:"character,omitempty"`
	Image          string  `json:"image"`                     // 画面描述
	Narration      string  `json:"narration"`                 // 旁白
	SoundEffect    string  `json:"sound_effect,omitempty"`    // 音效描述
	Duration       float64 `json:"duration,omitempty"`        // 时长（秒）
	ImagePrompt    string  `json:"image_prompt"`              // 镜头图片提示词
	VideoPrompt    string  `json:"video_prompt"`              // 镜头视频提示词
	CameraMovement string  `json:"camera_movement,omitempty"` // 运镜方式

	Sequence int    `json:"sequence"`
	Index    int    `json:"index"`
	Version  int    `json:"version"`
	Status   string `json:"status"`

	ErrorMessage string `json:"error_message,omitempty"` // 错误信息（失败时）

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toShotInfo(s *novel.Shot) ShotInfo {
	return ShotInfo{
		ID:             s.ID,
		SceneID:        s.SceneID,
		SceneNumber:    s.SceneNumber,
		NarrationID:    s.NarrationID,
		ChapterID:      s.ChapterID,
		UserID:         s.UserID,
		ShotNumber:     s.ShotNumber,
		Character:      s.Character,
		Image:          s.Image,
		Narration:      s.Narration,
		SoundEffect:    s.SoundEffect,
		Duration:       s.Duration,
		ImagePrompt:    s.ImagePrompt,
		VideoPrompt:    s.VideoPrompt,
		CameraMovement: s.CameraMovement,
		Sequence:       s.Sequence,
		Index:          s.Index,
		Version:        s.Version,
		Status:         string(s.Status),
		ErrorMessage:   s.ErrorMessage,
		CreatedAt:      s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      s.UpdatedAt.Format(time.RFC3339),
	}
}

// GetShotsByNarration 获取解说的镜头列表
// @Summary      获取解说镜头列表
// @Description  根据 narration_id 获取镜头列表（按 index 排序）
// @Tags         解说管理
// @Accept       json
// @Produce      json
// @Param        narration_id  path      string  true  "解说ID"
// @Success      200           {object}  map[string]interface{}  "成功响应"
// @Failure      400           {object}  ErrorResponse          "请求参数错误"
// @Failure      500           {object}  ErrorResponse          "服务器内部错误"
// @Router       /api/v1/narrations/{narration_id}/shots [get]
func (h *Handler) GetShotsByNarration(c *gin.Context) {
	narrationID := c.Param("narration_id")
	if narrationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "narration_id is required",
		})
		return
	}

	ctx := c.Request.Context()
	shots, err := h.novelService.GetShotsByNarrationID(ctx, narrationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	infos := make([]ShotInfo, 0, len(shots))
	for _, s := range shots {
		infos = append(infos, toShotInfo(s))
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"narration_id": narrationID,
			"shots":        infos,
			"count":        len(infos),
		},
	})
}
