package novel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/service"
)

// GenerateAudiosForNarration 为解说文案生成所有音频片段
// 参考 Python 的 gen_audio.py 逻辑
//
// Args:
//   - ctx: 上下文
//   - narrationID: 解说文案ID
//
// Returns:
//   - []string: 生成的音频ID列表
//   - error: 错误信息
func (s *novelService) GenerateAudiosForNarration(ctx context.Context, narrationID string) ([]string, error) {
	// 1. 从数据库获取解说文案
	narration, err := s.narrationRepo.FindByID(ctx, narrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find narration: %w", err)
	}

	if narration.Content == nil {
		return nil, fmt.Errorf("narration content is nil")
	}

	// 2. 从 JSON content 中提取所有解说文本
	extractor := noveltools.NewNarrationExtractor()
	narrationTexts, err := extractor.ExtractNarrationTexts(narration.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract narration texts: %w", err)
	}

	if len(narrationTexts) == 0 {
		return nil, fmt.Errorf("no narration texts found")
	}

	// 3. 为每段解说文本生成音频
	textCleaner := noveltools.NewTextCleaner()
	var audioIDs []string
	for i, narrationText := range narrationTexts {
		sequence := i + 1

		// 清理文本用于TTS
		cleanText := textCleaner.CleanTextForTTS(narrationText)
		if cleanText == "" {
			log.Warn().Int("sequence", sequence).Msg("清理后的文本为空，跳过")
			continue
		}

		// 生成音频
		audioID, err := s.generateSingleAudio(ctx, narration, sequence, cleanText)
		if err != nil {
			log.Error().Err(err).Int("sequence", sequence).Msg("生成音频失败")
			return nil, fmt.Errorf("failed to generate audio for sequence %d: %w", sequence, err)
		}

		audioIDs = append(audioIDs, audioID)
	}

	return audioIDs, nil
}

// generateSingleAudio 生成单个音频片段
func (s *novelService) generateSingleAudio(
	ctx context.Context,
	narration *novel.Narration,
	sequence int,
	text string,
) (string, error) {
	// 1. 创建临时文件用于保存音频
	tmpDir := os.TempDir()
	tmpAudioPath := filepath.Join(tmpDir, fmt.Sprintf("audio_%s_%d_%d.mp3", narration.ID, sequence, time.Now().Unix()))
	defer os.Remove(tmpAudioPath) // 清理临时文件

	// 2. 调用 TTS Provider 生成音频（1.2倍速，参考 Python 脚本）
	speedRatio := 1.2
	ttsResult, err := s.ttsProvider.GenerateVoiceWithTimestamps(ctx, text, tmpAudioPath, speedRatio)
	if err != nil {
		return "", fmt.Errorf("TTS generation failed: %w", err)
	}

	if !ttsResult.Success {
		return "", fmt.Errorf("TTS generation failed: %s", ttsResult.ErrorMessage)
	}

	// 构建 TTS 参数提示词（记录生成参数）
	ttsPrompt := fmt.Sprintf("TTS参数: speedRatio=%.2f, textLength=%d", speedRatio, len(text))

	// 3. 读取生成的音频文件
	audioFile, err := os.Open(tmpAudioPath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer audioFile.Close()

	// 4. 通过 resource 模块上传音频文件
	userID := narration.UserID
	fileName := fmt.Sprintf("%s_audio_%02d.mp3", narration.ID, sequence)
	contentType := "audio/mpeg"
	ext := "mp3"

	_, err = audioFile.Seek(0, 0)
	if err != nil {
		return "", fmt.Errorf("failed to seek audio file: %w", err)
	}

	// 使用 resource 模块上传文件
	uploadReq := &service.UploadFileRequest{
		UserID:      userID,
		FileName:    fileName,
		ContentType: contentType,
		Ext:         ext,
		Data:        audioFile,
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to upload audio file via resource service: %w", err)
	}

	resourceID := uploadResult.ResourceID

	// 6. 转换时间戳数据
	charTimes := make([]novel.CharTime, 0, len(ttsResult.TimestampData.CharacterTimestamps))
	for _, ts := range ttsResult.TimestampData.CharacterTimestamps {
		charTimes = append(charTimes, novel.CharTime{
			Character: ts.Character,
			StartTime: ts.StartTime,
			EndTime:   ts.EndTime,
		})
	}

	// 7. 创建 audio 记录
	audioID := id.New()
	audioEntity := &novel.Audio{
		ID:              audioID,
		NarrationID:     narration.ID,
		ChapterID:       narration.ChapterID,
		UserID:          narration.UserID,
		Sequence:        sequence,
		AudioResourceID: resourceID,
		Duration:        ttsResult.TimestampData.Duration,
		Text:            text,
		Timestamps:      charTimes,
		Prompt:          ttsPrompt,
		Status:          "completed",
	}

	if err := s.audioRepo.Create(ctx, audioEntity); err != nil {
		return "", fmt.Errorf("failed to create audio record: %w", err)
	}

	return audioID, nil
}
