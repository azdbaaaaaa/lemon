package novel

import (
	"context"
	"fmt"

	"lemon/internal/model/novel"
)

// ListAudiosByNarration 获取解说的音频列表（可指定版本；version<=0 则取最新版本）
func (s *novelService) ListAudiosByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Audio, int, error) {
	v, err := s.resolveAudioVersion(ctx, narrationID, version)
	if err != nil {
		return nil, 0, err
	}
	audios, err := s.audioRepo.FindByNarrationIDAndVersion(ctx, narrationID, v)
	if err != nil {
		return nil, 0, err
	}
	return audios, v, nil
}

// ListSubtitlesByNarration 获取解说的字幕列表（可指定版本；version<=0 则取最新版本）
func (s *novelService) ListSubtitlesByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Subtitle, int, error) {
	v, err := s.resolveSubtitleVersion(ctx, narrationID, version)
	if err != nil {
		return nil, 0, err
	}
	subs, err := s.subtitleRepo.FindByNarrationIDAndVersion(ctx, narrationID, v)
	if err != nil {
		return nil, 0, err
	}
	return subs, v, nil
}

// ListImagesByNarration 获取解说的图片列表（可指定版本；version<=0 则取最新版本）
func (s *novelService) ListImagesByNarration(ctx context.Context, narrationID string, version int) ([]*novel.Image, int, error) {
	v, err := s.resolveImageVersion(ctx, narrationID, version)
	if err != nil {
		return nil, 0, err
	}
	images, err := s.imageRepo.FindByNarrationIDAndVersion(ctx, narrationID, v)
	if err != nil {
		return nil, 0, err
	}
	return images, v, nil
}

// ListVideosByChapter 获取章节视频列表（可指定版本；version<=0 则取最新版本）
func (s *novelService) ListVideosByChapter(ctx context.Context, chapterID string, version int) ([]*novel.Video, int, error) {
	v, err := s.resolveVideoVersion(ctx, chapterID, version)
	if err != nil {
		return nil, 0, err
	}
	videos, err := s.videoRepo.FindByChapterIDAndVersion(ctx, chapterID, v)
	if err != nil {
		return nil, 0, err
	}
	return videos, v, nil
}

func (s *novelService) resolveAudioVersion(ctx context.Context, narrationID string, version int) (int, error) {
	if version > 0 {
		return version, nil
	}
	versions, err := s.audioRepo.FindVersionsByNarrationID(ctx, narrationID)
	if err != nil {
		return 0, err
	}
	return maxInt(versions)
}

func (s *novelService) resolveSubtitleVersion(ctx context.Context, narrationID string, version int) (int, error) {
	if version > 0 {
		return version, nil
	}
	subs, err := s.subtitleRepo.FindByNarrationID(ctx, narrationID)
	if err != nil {
		return 0, err
	}
	versions := make([]int, 0, len(subs))
	for _, s := range subs {
		versions = append(versions, s.Version)
	}
	return maxInt(versions)
}

func (s *novelService) resolveImageVersion(ctx context.Context, narrationID string, version int) (int, error) {
	if version > 0 {
		return version, nil
	}
	images, err := s.imageRepo.FindByNarrationID(ctx, narrationID)
	if err != nil {
		return 0, err
	}
	versions := make([]int, 0, len(images))
	for _, im := range images {
		versions = append(versions, im.Version)
	}
	return maxInt(versions)
}

func (s *novelService) resolveVideoVersion(ctx context.Context, chapterID string, version int) (int, error) {
	if version > 0 {
		return version, nil
	}
	versions, err := s.videoRepo.FindVersionsByChapterID(ctx, chapterID)
	if err != nil {
		return 0, err
	}
	return maxInt(versions)
}

func maxInt(values []int) (int, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("no versions found")
	}
	maxV := 0
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	if maxV == 0 {
		return 0, fmt.Errorf("no valid versions found")
	}
	return maxV, nil
}


