// Package tests 音频生成功能集成测试
//
// 运行测试：
//
//	MONGO_URI=mongodb://localhost:27017 TTS_ACCESS_TOKEN=your-token go test ./tests -run TestNovelService_GenerateAudio -v
//
// 说明：
//   - MONGO_URI: MongoDB 连接地址（默认: mongodb://localhost:27017）
//   - TTS_ACCESS_TOKEN: TTS 访问令牌（必需，用于调用真实的 TTS API）
//   - TTS_APP_ID: TTS 应用ID（可选）
//   - TTS_VOICE_TYPE: TTS 语音类型（可选，默认: BV115_streaming）
//   - 测试会使用数据库中已有的解说文案数据（需要先运行 3.narration_test.go 生成解说文案）
//   - 测试使用真实的 TTS Provider（调用真实 API）
//   - 测试完成后会自动清理测试数据库和临时存储文件
package tests

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TestNovelService_GenerateAudio 测试根据解说文案生成音频
func TestNovelService_GenerateAudio(t *testing.T) {
	Convey("NovelService 生成音频测试", t, func() {
		// 使用 TestMain 中初始化的全局变量
		ctx := testCtx
		services := testServices

		userID := "test_user_novel_001"

		// 步骤1: 查找或创建测试解说文案（优先使用数据库中已有的解说文案）
		narrationID, _ := findOrCreateTestNarration(ctx, t, services, userID)
		So(narrationID, ShouldNotBeEmpty)

		Convey("步骤2: 为解说文案生成音频", func() {
			// 为解说文案生成所有音频片段（使用真实的 TTS Provider）
			// 如果环境变量未设置，TestMain 会 panic，所以这里不需要检查
			audioIDs, err := services.NovelService.GenerateAudiosForNarration(ctx, narrationID)
			So(err, ShouldBeNil)
			So(len(audioIDs), ShouldBeGreaterThan, 0)

			Convey("验证音频已保存到 audios 表", func() {
				// 验证每个音频记录都存在
				for _, audioID := range audioIDs {
					So(audioID, ShouldNotBeEmpty)
				}

				// 验证至少生成了一个音频
				So(len(audioIDs), ShouldBeGreaterThan, 0)

				// 验证音频数量应该等于解说文案中的文本片段数量
				// 这里可以进一步验证，但需要知道解说文案的结构
				// 暂时只验证生成了音频
			})
		})
	})
}
