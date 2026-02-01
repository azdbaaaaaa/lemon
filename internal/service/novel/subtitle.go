package novel

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/noveltools"
	"lemon/internal/service"
)

// GenerateSubtitlesForNarration 为解说文案生成字幕文件（ASS格式）
// 参考 Python 的 gen_ass.py 逻辑
//
// Args:
//   - ctx: 上下文
//   - narrationID: 解说文案ID
//
// Returns:
//   - string: 生成的字幕ID
//   - error: 错误信息
func (s *novelService) GenerateSubtitlesForNarration(ctx context.Context, narrationID string) (string, error) {
	// 1. 从数据库获取解说文案
	narration, err := s.narrationRepo.FindByID(ctx, narrationID)
	if err != nil {
		return "", fmt.Errorf("failed to find narration: %w", err)
	}

	if narration.Content == nil {
		return "", fmt.Errorf("narration content is nil")
	}

	// 2. 获取该解说文案的所有音频记录（需要时间戳数据）
	audios, err := s.audioRepo.FindByNarrationID(ctx, narrationID)
	if err != nil {
		return "", fmt.Errorf("failed to find audios: %w", err)
	}

	if len(audios) == 0 {
		return "", fmt.Errorf("no audio records found for narration %s, please generate audio first", narrationID)
	}

	// 3. 从解说文案中提取所有文本
	extractor := noveltools.NewNarrationExtractor()
	narrationTexts, err := extractor.ExtractNarrationTexts(narration.Content)
	if err != nil {
		return "", fmt.Errorf("failed to extract narration texts: %w", err)
	}

	if len(narrationTexts) == 0 {
		return "", fmt.Errorf("no narration texts found")
	}

	// 4. 合并所有音频的时间戳数据（按顺序）
	allCharacterTimestamps := make([]noveltools.CharTimestamp, 0)
	allTexts := make([]string, 0)
	timeOffset := 0.0
	for i, audio := range audios {
		if len(audio.Timestamps) == 0 {
			log.Warn().Str("audio_id", audio.ID).Int("sequence", audio.Sequence).Msg("音频记录缺少时间戳数据，跳过")
			// 即使没有时间戳，也要累加时长，以便后续音频的时间戳正确偏移
			timeOffset += audio.Duration
			continue
		}

		// 转换并添加时间偏移
		for _, charTime := range audio.Timestamps {
			allCharacterTimestamps = append(allCharacterTimestamps, noveltools.CharTimestamp{
				Character: charTime.Character,
				StartTime: charTime.StartTime + timeOffset,
				EndTime:   charTime.EndTime + timeOffset,
			})
		}

		// 累加当前音频的时长，用于下一个音频的时间偏移
		timeOffset += audio.Duration

		// 添加对应的文本
		if i < len(narrationTexts) {
			allTexts = append(allTexts, narrationTexts[i])
		}
	}

	if len(allCharacterTimestamps) == 0 {
		return "", fmt.Errorf("no character timestamps found in audio records")
	}

	// 5. 合并所有文本
	combinedText := strings.Join(allTexts, "")

	// 6. 使用 SubtitleSplitter 分割文本（默认每段最大12字符）
	maxLength := 12
	splitter := noveltools.NewSubtitleSplitter(maxLength)
	segments := splitter.SplitTextNaturally(combinedText)

	if len(segments) == 0 {
		return "", fmt.Errorf("no segments found after splitting text")
	}

	// 7. 使用 SubtitleTimestampCalculator 计算时间戳
	calculator := noveltools.NewSubtitleTimestampCalculator()
	segmentTimestamps := calculator.CalculateSegmentTimestamps(
		segments,
		allCharacterTimestamps,
		combinedText,
	)

	if len(segmentTimestamps) == 0 {
		return "", fmt.Errorf("failed to calculate segment timestamps")
	}

	// 8. 使用 ASSGenerator 生成 ASS 内容
	assGenerator := noveltools.NewASSGenerator()
	var assContent string
	chapter, err := s.chapterRepo.FindByID(ctx, narration.ChapterID)
	if err == nil {
		title := fmt.Sprintf("Chapter %s Narration Subtitle", chapter.Title)
		assContent = assGenerator.GenerateASSContent(segmentTimestamps, title)
	} else {
		assContent = assGenerator.GenerateASSContent(segmentTimestamps, "Generated Subtitle")
	}

	// 9. 直接在内存中创建 ASS 文件的 io.Reader（无需临时文件）
	assContentBytes := []byte(assContent)
	assReader := bytes.NewReader(assContentBytes)

	// 10. 通过 resource 模块上传 ASS 文件
	userID := narration.UserID
	fileName := fmt.Sprintf("%s_subtitle.ass", narration.ID)
	contentType := "text/x-ass"
	ext := "ass"

	// 使用 resource 模块上传文件
	uploadReq := &service.UploadFileRequest{
		UserID:      userID,
		FileName:    fileName,
		ContentType: contentType,
		Ext:         ext,
		Data:        assReader,
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to upload ASS file via resource service: %w", err)
	}

	resourceID := uploadResult.ResourceID

	// 12. 构建字幕生成参数提示词（记录生成参数）
	subtitlePrompt := fmt.Sprintf("字幕生成参数: maxLength=%d, format=ass, segmentCount=%d", maxLength, len(segmentTimestamps))

	// 13. 创建 subtitle 记录
	subtitleID := id.New()
	subtitleEntity := &novel.Subtitle{
		ID:                 subtitleID,
		ChapterID:          narration.ChapterID,
		NarrationID:        narration.ID,
		UserID:             narration.UserID,
		SubtitleResourceID: resourceID,
		Format:             "ass",
		Prompt:             subtitlePrompt,
		Status:             "completed",
	}

	if err := s.subtitleRepo.Create(ctx, subtitleEntity); err != nil {
		return "", fmt.Errorf("failed to create subtitle record: %w", err)
	}

	return subtitleID, nil
}
