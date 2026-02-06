// Package tests 小说创建和章节切分功能集成测试
//
// 运行测试：
//
//	MONGO_URI=mongodb://localhost:27017 go test ./tests -run TestNovelService_CreateAndSplit -v
//
// 说明：
//   - MONGO_URI: MongoDB 连接地址（默认: mongodb://localhost:27017）
//   - 测试会读取 assets/novel/001/大道主(飘荡的云).txt 作为测试数据
//   - 测试使用本地文件系统存储（临时目录）
//   - 测试完成后会自动清理测试数据库和临时存储文件
package tests

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"lemon/internal/model/novel"
)

// TestNovelService_CreateAndSplit 测试根据上传的资源创建小说并切分章节
func TestNovelService_CreateAndSplit(t *testing.T) {
	Convey("NovelService 创建小说和切分章节测试", t, func() {
		// 使用 TestMain 中初始化的全局变量
		ctx := testCtx
		services := testServices

		userID := "test_user_novel_001"

		// 步骤1: 查找或上传测试文件（优先使用数据库中已有的资源）
		resourceID := findOrUploadTestFile(ctx, t, services, userID)

		Convey("步骤2: 根据资源创建小说", func() {
			novelID, err := services.NovelService.CreateNovelFromResource(ctx, resourceID, userID, novel.NarrationTypeNarration, novel.NovelStyleAnime)
			So(err, ShouldBeNil)
			So(novelID, ShouldNotBeEmpty)

			// 验证小说记录存在
			novelEntity, err := services.NovelService.GetNovel(ctx, novelID)
			So(err, ShouldBeNil)
			So(novelEntity.ResourceID, ShouldEqual, resourceID)
			So(novelEntity.UserID, ShouldEqual, userID)

			Convey("步骤3: 切分章节", func() {
				targetChapters := 50
				err := services.NovelService.SplitNovelIntoChapters(ctx, novelID, targetChapters)
				So(err, ShouldBeNil)

				Convey("验证章节切分结果", func() {
					// 验证章节已创建
					chapters, err := services.NovelService.GetChapters(ctx, novelID)
					So(err, ShouldBeNil)
					So(len(chapters), ShouldBeGreaterThan, 0)
					So(len(chapters), ShouldBeLessThanOrEqualTo, targetChapters+10) // 允许一些误差

					// 验证章节顺序和内容
					for i, ch := range chapters {
						So(ch.Sequence, ShouldEqual, i+1)
						So(ch.NovelID, ShouldEqual, novelID)
						So(ch.UserID, ShouldEqual, userID)
						So(ch.Title, ShouldNotBeEmpty)
						So(ch.ChapterText, ShouldNotBeEmpty)
						So(len([]rune(ch.ChapterText)), ShouldBeGreaterThan, 0)
					}

					// 验证总字数
					totalChars := 0
					for _, ch := range chapters {
						totalChars += len([]rune(ch.ChapterText))
					}
					So(totalChars, ShouldBeGreaterThan, 0)
					So(totalChars, ShouldBeGreaterThan, 1000)

					// 验证章节内容不重复（简单检查：前几个字符不应该完全相同）
					if len(chapters) > 1 {
						firstChapterStart := []rune(chapters[0].ChapterText)[:min(50, len([]rune(chapters[0].ChapterText)))]
						secondChapterStart := []rune(chapters[1].ChapterText)[:min(50, len([]rune(chapters[1].ChapterText)))]
						So(string(firstChapterStart), ShouldNotEqual, string(secondChapterStart))
					}

					// 验证章节标题格式（应该包含章节序号或标题）
					for _, ch := range chapters {
						So(ch.Title, ShouldNotBeEmpty)
						// 标题可能包含"第X章"、"Chapter X"等格式，这里只验证不为空
					}
				})
			})
		})
	})
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
