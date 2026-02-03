package noveltools

import (
	"fmt"

	"lemon/internal/model/novel"
)

// NarrationExtractor 解说内容提取器
// 用于从解说文案内容中提取所有解说文本
type NarrationExtractor struct{}

// NewNarrationExtractor 创建解说内容提取器实例
func NewNarrationExtractor() *NarrationExtractor {
	return &NarrationExtractor{}
}

// ExtractNarrationTexts 从解说文案内容中提取所有解说文本
// 参考 Python 的 extract_narration_content，但适配结构体格式
//
// Args:
//   - content: 解说文案的 Content 字段（*NarrationContent）
//
// Returns:
//   - []string: 所有解说文本列表（按顺序）
//   - error: 错误信息
func (ne *NarrationExtractor) ExtractNarrationTexts(content *novel.NarrationContent) ([]string, error) {
	if content == nil {
		return nil, fmt.Errorf("content is nil")
	}

	if len(content.Scenes) == 0 {
		return nil, fmt.Errorf("scenes field is missing or empty")
	}

	var narrationTexts []string

	for _, scene := range content.Scenes {
		if scene == nil {
			continue
		}

		// 只提取特写级别的解说内容（每个 shot 对应一个分镜）
		// 不提取 scene.Narration，因为每个分镜应该只对应一个 shot.Narration
		if scene.Shots != nil {
			for _, shot := range scene.Shots {
				if shot == nil {
					continue
				}
				if shot.Narration != "" {
					narrationTexts = append(narrationTexts, shot.Narration)
				}
			}
		}
	}

	if len(narrationTexts) == 0 {
		return nil, fmt.Errorf("no narration texts found in content")
	}

	return narrationTexts, nil
}
