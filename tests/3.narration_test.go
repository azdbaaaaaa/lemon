// Package tests 章节解说文案生成功能集成测试
//
// 运行测试：
//
//	MONGO_URI=mongodb://localhost:27017 go test ./tests -run TestNovelService_GenerateNarration -v
//
// 说明：
//   - MONGO_URI: MongoDB 连接地址（默认: mongodb://localhost:27017）
//   - ARK_API_KEY: Ark API Key（必需，用于调用真实的 LLM API）
//   - ARK_MODEL: 模型名称（可选，默认: doubao-seed-1-6-flash-250615）
//   - ARK_BASE_URL: API 基础 URL（可选，默认: https://ark.cn-beijing.volces.com/api/v3）
//   - 测试会使用数据库中已有的章节数据（需要先运行 2.novel_test.go 创建章节）
//   - 测试使用真实的 LLM Provider（调用真实 API）
//   - 测试完成后会自动清理测试数据库和临时存储文件
package tests

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TestNovelService_GenerateNarrationText 测试根据章节生成解说文案
func TestNovelService_GenerateNarrationText(t *testing.T) {
	Convey("NovelService 生成章节解说文案测试", t, func() {
		// 使用 TestMain 中初始化的全局变量
		ctx := testCtx
		services := testServices

		userID := "test_user_novel_001"

		// 步骤1: 查找或创建测试章节（优先使用数据库中已有的章节）
		novelID, chapters := findOrCreateTestChapters(ctx, t, services, userID)
		So(novelID, ShouldNotBeEmpty)
		So(len(chapters), ShouldBeGreaterThan, 0)

		Convey("步骤2: 为章节生成解说文案", func() {
			// 为第一个章节生成解说文案（使用真实的 LLM Provider）
			// 如果环境变量未设置，TestMain 会 panic，所以这里不需要检查
			firstChapter := chapters[0]
			narrationText, err := services.NovelService.GenerateNarrationForChapter(ctx, firstChapter.ID)
			So(err, ShouldBeNil)
			So(narrationText, ShouldNotBeEmpty)

			Convey("验证解说文案已保存到 narrations 表", func() {
				// 查询 narrations 表，验证解说文案已保存
				narrationEntity, err := services.NovelService.GetNarration(ctx, firstChapter.ID)
				So(err, ShouldBeNil)
				So(narrationEntity.ChapterID, ShouldEqual, firstChapter.ID)
				So(narrationEntity.Status, ShouldEqual, "completed")

				// 验证包含必要的字段
				So(len(narrationEntity.Content.Scenes), ShouldBeGreaterThan, 0)
			})
		})
	})
}
