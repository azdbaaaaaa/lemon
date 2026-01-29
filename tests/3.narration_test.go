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
	"encoding/json"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TestNovelService_GenerateNarration 测试根据章节生成解说文案
func TestNovelService_GenerateNarration(t *testing.T) {
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
			// 检查是否有 LLM Provider
			if services.LLMProvider == nil {
				// 检查环境变量是否设置（用于调试）
				apiKey := os.Getenv("ARK_API_KEY")
				if apiKey != "" {
					t.Fatalf("ARK_API_KEY 已设置但 LLM Provider 为 nil，可能是初始化失败。请检查 TestMain 的错误日志。")
				}
				t.Skip("跳过测试：ARK_API_KEY 未设置，无法使用真实的 LLM Provider")
			}

			// 为第一个章节生成解说文案（使用真实的 LLM Provider）
			firstChapter := chapters[0]
			narrationText, err := services.NovelService.GenerateNarrationForChapter(ctx, firstChapter.ID)
			So(err, ShouldBeNil)
			So(narrationText, ShouldNotBeEmpty)

			Convey("验证解说文案已保存到 narrations 表", func() {
				// 查询 narrations 表，验证解说文案已保存
				narrationEntity, err := services.NarrationRepo.FindByChapterID(ctx, firstChapter.ID)
				So(err, ShouldBeNil)
				So(narrationEntity, ShouldNotBeNil)
				So(narrationEntity.ChapterID, ShouldEqual, firstChapter.ID)
				So(narrationEntity.Content, ShouldNotBeNil)
				So(narrationEntity.Status, ShouldEqual, "completed")

				// 验证内容为结构化数据（map）
				So(narrationEntity.Content, ShouldNotBeNil)

				// 验证包含必要的字段
				So(narrationEntity.Content["scenes"], ShouldNotBeNil)

				// MongoDB 可能返回不同的类型，需要更灵活的类型检查
				scenesValue := narrationEntity.Content["scenes"]
				So(scenesValue, ShouldNotBeNil)

				// 尝试多种类型断言
				var scenes []interface{}
				switch v := scenesValue.(type) {
				case []interface{}:
					scenes = v
				case []map[string]interface{}:
					// 转换为 []interface{}
					scenes = make([]interface{}, len(v))
					for i, item := range v {
						scenes[i] = item
					}
				default:
					// 尝试通过 JSON 序列化/反序列化来转换类型
					jsonBytes, _ := json.Marshal(scenesValue)
					json.Unmarshal(jsonBytes, &scenes)
				}

				So(len(scenes), ShouldBeGreaterThan, 0)
			})

			Convey("为多个章节生成解说文案", func() {
				// 为前2个章节生成解说文案（减少 API 调用次数）
				chaptersToTest := chapters
				if len(chapters) > 2 {
					chaptersToTest = chapters[:2]
				}

				for _, ch := range chaptersToTest {
					narrationText, err := services.NovelService.GenerateNarrationForChapter(ctx, ch.ID)
					So(err, ShouldBeNil)
					So(narrationText, ShouldNotBeEmpty)

					// 验证已保存到 narrations 表
					narrationEntity, err := services.NarrationRepo.FindByChapterID(ctx, ch.ID)
					So(err, ShouldBeNil)
					So(narrationEntity, ShouldNotBeNil)
					So(narrationEntity.Content, ShouldNotBeNil)

					// 验证内容为结构化数据
					So(narrationEntity.Content, ShouldNotBeNil)

					// 验证 scenes 字段存在（使用灵活的类型检查）
					scenesValue := narrationEntity.Content["scenes"]
					So(scenesValue, ShouldNotBeNil)

					// 尝试多种类型断言
					var scenes []interface{}
					switch v := scenesValue.(type) {
					case []interface{}:
						scenes = v
					case []map[string]interface{}:
						// 转换为 []interface{}
						scenes = make([]interface{}, len(v))
						for i, item := range v {
							scenes[i] = item
						}
					default:
						// 尝试通过 JSON 序列化/反序列化来转换类型
						jsonBytes, _ := json.Marshal(scenesValue)
						json.Unmarshal(jsonBytes, &scenes)
					}
					So(len(scenes), ShouldBeGreaterThan, 0)
				}
			})
		})
	})
}
