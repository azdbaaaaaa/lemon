package novel

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"lemon/internal/model/novel"
)

// AudioInfo 音频信息 DTO
type AudioInfo struct {
	ID              string  `json:"id"`
	NarrationID     string  `json:"narration_id"`
	ChapterID       string  `json:"chapter_id"`
	UserID          string  `json:"user_id"`
	Sequence        int     `json:"sequence"`
	AudioResourceID string  `json:"audio_resource_id"`
	Duration        float64 `json:"duration"`
	Text            string  `json:"text"`
	Prompt          string  `json:"prompt,omitempty"`
	Version         int     `json:"version"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func toAudioInfo(a *novel.Audio) AudioInfo {
	return AudioInfo{
		ID:              a.ID,
		NarrationID:     a.NarrationID,
		ChapterID:       a.ChapterID,
		UserID:          a.UserID,
		Sequence:        a.Sequence,
		AudioResourceID: a.AudioResourceID,
		Duration:        a.Duration,
		Text:            a.Text,
		Prompt:          a.Prompt,
		Version:         a.Version,
		Status:          string(a.Status),
		CreatedAt:       a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       a.UpdatedAt.Format(time.RFC3339),
	}
}

// SubtitleInfo 字幕信息 DTO
type SubtitleInfo struct {
	ID                 string `json:"id"`
	ChapterID          string `json:"chapter_id"`
	NarrationID        string `json:"narration_id"`
	UserID             string `json:"user_id"`
	Sequence           int    `json:"sequence"`
	SubtitleResourceID string `json:"subtitle_resource_id"`
	Format             string `json:"format"`
	Prompt             string `json:"prompt,omitempty"`
	Version            int    `json:"version"`
	Status             string `json:"status"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

func toSubtitleInfo(s *novel.Subtitle) SubtitleInfo {
	return SubtitleInfo{
		ID:                 s.ID,
		ChapterID:          s.ChapterID,
		NarrationID:        s.NarrationID,
		UserID:             s.UserID,
		Sequence:           s.Sequence,
		SubtitleResourceID: s.SubtitleResourceID,
		Format:             string(s.Format),
		Prompt:             s.Prompt,
		Version:            s.Version,
		Status:             string(s.Status),
		CreatedAt:          s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          s.UpdatedAt.Format(time.RFC3339),
	}
}

// ImageInfo 图片信息 DTO
type ImageInfo struct {
	ID              string `json:"id"`
	ChapterID        string `json:"chapter_id"`
	NarrationID      string `json:"narration_id"`
	SceneNumber      string `json:"scene_number"`
	ShotNumber       string `json:"shot_number"`
	ImageResourceID  string `json:"image_resource_id"`
	CharacterName    string `json:"character_name"`
	Prompt           string `json:"prompt,omitempty"`
	Version          int    `json:"version"`
	Status           string `json:"status"`
	Sequence         int    `json:"sequence"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func toImageInfo(i *novel.Image) ImageInfo {
	return ImageInfo{
		ID:             i.ID,
		ChapterID:       i.ChapterID,
		NarrationID:     i.NarrationID,
		SceneNumber:     i.SceneNumber,
		ShotNumber:      i.ShotNumber,
		ImageResourceID: i.ImageResourceID,
		CharacterName:   i.CharacterName,
		Prompt:          i.Prompt,
		Version:         i.Version,
		Status:          string(i.Status),
		Sequence:        i.Sequence,
		CreatedAt:       i.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       i.UpdatedAt.Format(time.RFC3339),
	}
}

// ListAudiosByNarration 列出解说的音频列表（可选 version）
// @Router /api/v1/narrations/{narration_id}/audios [get]
func (h *Handler) ListAudiosByNarration(c *gin.Context) {
	narrationID := c.Param("narration_id")
	if narrationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 40001, Message: "narration_id is required"})
		return
	}
	version := parseOptionalIntQuery(c, "version")

	ctx := c.Request.Context()
	audios, resolved, err := h.novelService.ListAudiosByNarration(ctx, narrationID, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 50001, Message: err.Error()})
		return
	}
	out := make([]AudioInfo, 0, len(audios))
	for _, a := range audios {
		out = append(out, toAudioInfo(a))
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"narration_id": narrationID,
			"version":      resolved,
			"audios":       out,
			"count":        len(out),
		},
	})
}

// ListSubtitlesByNarration 列出解说的字幕列表（可选 version）
// @Router /api/v1/narrations/{narration_id}/subtitles [get]
func (h *Handler) ListSubtitlesByNarration(c *gin.Context) {
	narrationID := c.Param("narration_id")
	if narrationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 40001, Message: "narration_id is required"})
		return
	}
	version := parseOptionalIntQuery(c, "version")

	ctx := c.Request.Context()
	subs, resolved, err := h.novelService.ListSubtitlesByNarration(ctx, narrationID, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 50001, Message: err.Error()})
		return
	}
	out := make([]SubtitleInfo, 0, len(subs))
	for _, s := range subs {
		out = append(out, toSubtitleInfo(s))
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"narration_id": narrationID,
			"version":      resolved,
			"subtitles":    out,
			"count":        len(out),
		},
	})
}

// ListImagesByNarration 列出解说的图片列表（可选 version）
// @Router /api/v1/narrations/{narration_id}/images [get]
func (h *Handler) ListImagesByNarration(c *gin.Context) {
	narrationID := c.Param("narration_id")
	if narrationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 40001, Message: "narration_id is required"})
		return
	}
	version := parseOptionalIntQuery(c, "version")

	ctx := c.Request.Context()
	images, resolved, err := h.novelService.ListImagesByNarration(ctx, narrationID, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 50001, Message: err.Error()})
		return
	}
	out := make([]ImageInfo, 0, len(images))
	for _, i := range images {
		out = append(out, toImageInfo(i))
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"narration_id": narrationID,
			"version":      resolved,
			"images":       out,
			"count":        len(out),
		},
	})
}

// ListVideosByChapter 列出章节视频列表（可选 version）
// @Router /api/v1/novels/chapters/{chapter_id}/videos [get]
func (h *Handler) ListVideosByChapter(c *gin.Context) {
	chapterID := c.Param("chapter_id")
	if chapterID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: 40001, Message: "chapter_id is required"})
		return
	}
	version := parseOptionalIntQuery(c, "version")

	ctx := c.Request.Context()
	videos, resolved, err := h.novelService.ListVideosByChapter(ctx, chapterID, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: 50001, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"chapter_id": chapterID,
			"version":    resolved,
			"videos":     toVideoInfoList(videos),
			"count":      len(videos),
		},
	})
}

func parseOptionalIntQuery(c *gin.Context, key string) int {
	val := c.Query(key)
	if val == "" {
		return 0
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}


