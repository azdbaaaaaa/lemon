package noveltools

import (
	"strings"
)

// SegmentTimestamp 段落时间戳
type SegmentTimestamp struct {
	Text      string  `json:"text"`       // 段落文本
	StartTime float64 `json:"start_time"` // 开始时间（秒）
	EndTime   float64 `json:"end_time"`   // 结束时间（秒）
}

// SubtitleTimestampCalculator 字幕时间戳计算器
type SubtitleTimestampCalculator struct{}

// NewSubtitleTimestampCalculator 创建字幕时间戳计算器实例
func NewSubtitleTimestampCalculator() *SubtitleTimestampCalculator {
	return &SubtitleTimestampCalculator{}
}

// CalculateSegmentTimestamps 为分割后的段落计算时间戳，确保不会出现重叠
// 参考 Python 脚本 gen_ass.py 的 calculate_segment_timestamps 函数
func (stc *SubtitleTimestampCalculator) CalculateSegmentTimestamps(
	segments []string,
	characterTimestamps []CharTimestamp,
	originalText string,
) []SegmentTimestamp {
	// 预处理：创建清理后文本到原始字符索引的映射
	cleanToOriginalMapping, cleanOriginalText := stc.buildCleanTextMapping(characterTimestamps)

	// 为每个段落计算时间戳
	segmentTimestamps := stc.calculateTimestampsForSegments(
		segments, characterTimestamps, cleanToOriginalMapping, cleanOriginalText)

	// 最终检查：确保所有时间戳都是递增的且无重叠
	return stc.fixOverlappingTimestamps(segmentTimestamps)
}

// buildCleanTextMapping 创建清理后文本到原始字符索引的映射
func (stc *SubtitleTimestampCalculator) buildCleanTextMapping(
	characterTimestamps []CharTimestamp,
) ([]int, string) {
	cleanToOriginalMapping := []int{}
	cleanOriginalText := ""

	for i, charData := range characterTimestamps {
		char := charData.Character
		if !isPunctuation(char) {
			cleanOriginalText += char
			cleanToOriginalMapping = append(cleanToOriginalMapping, i)
		}
	}

	return cleanToOriginalMapping, cleanOriginalText
}

// calculateTimestampsForSegments 为段落计算时间戳
func (stc *SubtitleTimestampCalculator) calculateTimestampsForSegments(
	segments []string,
	characterTimestamps []CharTimestamp,
	cleanToOriginalMapping []int,
	cleanOriginalText string,
) []SegmentTimestamp {
	segmentTimestamps := []SegmentTimestamp{}
	currentCharIndex := 0

	for _, segment := range segments {
		cleanSegment := cleanSubtitleText(segment)
		startTime, endTime, newIndex := stc.findSegmentTimestamps(
			cleanSegment, characterTimestamps, cleanToOriginalMapping,
			cleanOriginalText, currentCharIndex, segmentTimestamps)

		currentCharIndex = newIndex

		// 检查并修正重叠问题
		startTime, endTime = stc.fixSegmentOverlap(startTime, endTime, segmentTimestamps, cleanSegment)

		segmentTimestamps = append(segmentTimestamps, SegmentTimestamp{
			Text:      segment,
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	return segmentTimestamps
}

// findSegmentTimestamps 查找段落的时间戳
func (stc *SubtitleTimestampCalculator) findSegmentTimestamps(
	cleanSegment string,
	characterTimestamps []CharTimestamp,
	cleanToOriginalMapping []int,
	cleanOriginalText string,
	currentCharIndex int,
	segmentTimestamps []SegmentTimestamp,
) (float64, float64, int) {
	// 在清理后的文本中查找段落位置
	segmentStartCleanIndex, segmentEndCleanIndex := stc.findSegmentInCleanText(
		cleanSegment, cleanOriginalText, currentCharIndex)

	if segmentStartCleanIndex == -1 || segmentEndCleanIndex == -1 {
		// 使用估算时间
		startTime := stc.estimateStartTime(segmentTimestamps)
		endTime := startTime + float64(len(cleanSegment))*0.3
		return startTime, endTime, currentCharIndex
	}

	// 映射回原始字符索引
	originalStartIndex, originalEndIndex := stc.mapToOriginalIndices(
		segmentStartCleanIndex, segmentEndCleanIndex, cleanToOriginalMapping)

	if originalStartIndex == -1 || originalEndIndex == -1 {
		startTime := stc.estimateStartTime(segmentTimestamps)
		endTime := startTime + float64(len(cleanSegment))*0.3
		return startTime, endTime, currentCharIndex
	}

	// 获取时间戳
	if originalStartIndex < len(characterTimestamps) &&
		originalEndIndex < len(characterTimestamps) {
		startTime := characterTimestamps[originalStartIndex].StartTime
		endTime := characterTimestamps[originalEndIndex].EndTime
		return startTime, endTime, segmentEndCleanIndex + 1
	}

	startTime := stc.estimateStartTime(segmentTimestamps)
	endTime := startTime + float64(len(cleanSegment))*0.3
	return startTime, endTime, currentCharIndex
}

// findSegmentInCleanText 在清理后的文本中查找段落位置
func (stc *SubtitleTimestampCalculator) findSegmentInCleanText(
	cleanSegment, cleanOriginalText string, currentCharIndex int,
) (int, int) {
	searchStart := currentCharIndex
	if searchStart >= len(cleanOriginalText) {
		searchStart = len(cleanOriginalText) - 1
	}
	if searchStart < 0 {
		searchStart = 0
	}

	for startPos := searchStart; startPos < len(cleanOriginalText); startPos++ {
		if startPos+len(cleanSegment) <= len(cleanOriginalText) {
			if cleanOriginalText[startPos:startPos+len(cleanSegment)] == cleanSegment {
				return startPos, startPos + len(cleanSegment) - 1
			}
		}
	}

	return -1, -1
}

// mapToOriginalIndices 将清理后的索引映射回原始字符索引
func (stc *SubtitleTimestampCalculator) mapToOriginalIndices(
	segmentStartCleanIndex, segmentEndCleanIndex int,
	cleanToOriginalMapping []int,
) (int, int) {
	if segmentStartCleanIndex < len(cleanToOriginalMapping) &&
		segmentEndCleanIndex < len(cleanToOriginalMapping) {
		return cleanToOriginalMapping[segmentStartCleanIndex],
			cleanToOriginalMapping[segmentEndCleanIndex]
	}
	return -1, -1
}

// estimateStartTime 估算开始时间
func (stc *SubtitleTimestampCalculator) estimateStartTime(segmentTimestamps []SegmentTimestamp) float64 {
	if len(segmentTimestamps) > 0 {
		return segmentTimestamps[len(segmentTimestamps)-1].EndTime + 0.1
	}
	return 0
}

// fixSegmentOverlap 修正段落重叠问题
func (stc *SubtitleTimestampCalculator) fixSegmentOverlap(
	startTime, endTime float64,
	segmentTimestamps []SegmentTimestamp,
	cleanSegment string,
) (float64, float64) {
	if len(segmentTimestamps) > 0 {
		prevEndTime := segmentTimestamps[len(segmentTimestamps)-1].EndTime
		if startTime < prevEndTime {
			startTime = prevEndTime + 0.1
			if startTime >= endTime {
				endTime = startTime + float64(len(cleanSegment))*0.3
			}
		} else if endTime <= startTime {
			endTime = startTime + float64(len(cleanSegment))*0.3
		}
	}
	return startTime, endTime
}

// fixOverlappingTimestamps 修正所有重叠的时间戳
func (stc *SubtitleTimestampCalculator) fixOverlappingTimestamps(
	segmentTimestamps []SegmentTimestamp,
) []SegmentTimestamp {
	for i := 1; i < len(segmentTimestamps); i++ {
		prevSegment := segmentTimestamps[i-1]
		currSegment := segmentTimestamps[i]

		if currSegment.StartTime < prevSegment.EndTime {
			newStartTime := prevSegment.EndTime + 0.1
			duration := currSegment.EndTime - currSegment.StartTime

			if duration < 0.5 {
				duration = 0.5
				cleanTextLen := len(cleanSubtitleText(currSegment.Text))
				if cleanTextLen*3 > 50 {
					duration = float64(cleanTextLen) * 0.3
				}
			}

			segmentTimestamps[i].StartTime = newStartTime
			segmentTimestamps[i].EndTime = newStartTime + duration
		}

		if segmentTimestamps[i].StartTime >= segmentTimestamps[i].EndTime {
			segmentTimestamps[i].EndTime = segmentTimestamps[i].StartTime + 1.0
		}
	}

	return segmentTimestamps
}

// isPunctuation 检查字符是否为标点符号
func isPunctuation(char string) bool {
	punctuation := "，。；：、！？\"\"''（）【】《》〈〉「」『』〔〕[]{}|～·…—–,.;:!?\"'()[]{}|~`@#$%^&*+=<>/\\-\npau"
	return len(char) == 1 && strings.Contains(punctuation, char)
}
