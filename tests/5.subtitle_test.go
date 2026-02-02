// Package tests 字幕生成功能集成测试
//
// 运行测试：
//
//	MONGO_URI=mongodb://localhost:27017 TTS_ACCESS_TOKEN=your-token go test ./tests -run TestNovelService_GenerateSubtitle -v
//
// 说明：
//   - MONGO_URI: MongoDB 连接地址（默认: mongodb://localhost:27017）
//   - TTS_ACCESS_TOKEN: TTS 访问令牌（必需，用于调用真实的 TTS API 生成音频）
//   - 测试会使用数据库中已有的解说文案和音频数据（需要先运行 3.narration_test.go 和 4.audio_test.go）
//   - 测试使用真实的 TTS Provider（调用真实 API）
//   - 测试完成后会自动清理测试数据库和临时存储文件
package tests

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TestNovelService_GenerateSubtitle 测试根据解说文案和音频生成字幕
func TestNovelService_GenerateSubtitle(t *testing.T) {
	Convey("NovelService 生成字幕测试", t, func() {
		// 使用 TestMain 中初始化的全局变量
		ctx := testCtx
		services := testServices

		userID := "test_user_novel_001"

		// 步骤1: 要求必须有测试解说文案，否则报错
		narrationID, _ := requireTestNarration(ctx, t, services, userID)
		So(narrationID, ShouldNotBeEmpty)

		// 步骤2: 要求必须有音频记录，否则报错
		requireTestAudios(ctx, t, narrationID)

		Convey("步骤3: 为解说文案生成字幕", func() {
			// 为解说文案生成字幕文件（ASS格式）
			subtitleID, err := services.NovelService.GenerateSubtitlesForNarration(ctx, narrationID)
			So(err, ShouldBeNil)
			So(subtitleID, ShouldNotBeEmpty)

			Convey("验证字幕已保存到 subtitles 表", func() {
				// 验证返回的 subtitleID 不为空
				So(subtitleID, ShouldNotBeEmpty)

				// 可以进一步验证：
				// 1. 字幕文件已上传到 resource 模块
				// 2. 字幕记录已保存到数据库
				// 3. 字幕格式为 ASS
				// 注意：目前 NovelService 没有 GetSubtitle 方法，所以无法直接验证数据库记录
				// 如果需要验证，可以添加 GetSubtitle 方法到 NovelService 接口
			})
		})
	})
}
