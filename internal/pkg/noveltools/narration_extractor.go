package noveltools

import (
	"fmt"
)

// NarrationExtractor 解说内容提取器
// 用于从解说文案内容中提取所有解说文本
// 注意：此函数已废弃，现在应该直接从 Shot 表查询数据
// 保留此函数仅用于向后兼容，但建议使用新的方式
type NarrationExtractor struct{}

// NewNarrationExtractor 创建解说内容提取器实例
func NewNarrationExtractor() *NarrationExtractor {
	return &NarrationExtractor{}
}

// ExtractNarrationTexts 从解说文案内容中提取所有解说文本
// 注意：此函数已废弃，现在应该直接从 Shot 表查询数据
// 保留此函数仅用于向后兼容，但建议使用新的方式
//
// Args:
//   - content: 解说文案的临时 JSON 结构体（*narrationJSONContent）
//
// Returns:
//   - []string: 所有解说文本列表（按顺序）
//   - error: 错误信息
func (ne *NarrationExtractor) ExtractNarrationTexts(content *NarrationJSONContent) ([]string, error) {
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
