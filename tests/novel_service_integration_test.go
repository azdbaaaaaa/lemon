// Package tests NovelService 集成测试
//
// 运行集成测试：
//
//	MONGO_URI=mongodb://localhost:27017 go test ./tests -run TestNovelService_Integration -v
//
// 说明：
//   - MONGO_URI: MongoDB 连接地址（默认: mongodb://localhost:27017）
//   - 测试会读取 assets/novel/001/大道主(飘荡的云).txt 作为测试数据
//   - 测试使用本地文件系统存储（临时目录）
//   - 测试完成后会自动清理测试数据库和临时存储文件
package tests

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"lemon/internal/pkg/id"
	narrationRepo "lemon/internal/repository/narration"
	novelrepo "lemon/internal/repository/novel"
	resourceRepo "lemon/internal/repository/resource"
	"lemon/internal/service"
)

func TestNovelService_Integration(t *testing.T) {
	Convey("NovelService 集成测试：完整的小说处理流程", t, func() {
		ctx, db, testStorage, cleanup := setupTestEnvironment(t)
		defer cleanup()

		// 初始化仓库
		resourceRepo := resourceRepo.NewResourceRepo(db)
		novelRepo := novelrepo.NewNovelRepo(db)
		chapterRepo := novelrepo.NewChapterRepo(db)
		narrationRepo := narrationRepo.NewNarrationRepo(db)

		// 初始化服务
		resourceService := service.NewResourceService(resourceRepo, testStorage)
		novelService := service.NewNovelService(resourceRepo, novelRepo, chapterRepo, narrationRepo, testStorage, nil)

		// 读取测试文件
		novelFilePath := getTestNovelFilePath(t)
		novelFile, err := os.Open(novelFilePath)
		So(err, ShouldBeNil, "应该能打开测试文件: "+novelFilePath)
		defer novelFile.Close()

		// 重置文件指针到开头（因为可能被其他操作移动过）
		_, err = novelFile.Seek(0, 0)
		So(err, ShouldBeNil)

		fileInfo, err := novelFile.Stat()
		So(err, ShouldBeNil)

		userID := "test_user_001"
		workflowID := id.New()

		Convey("步骤1: 上传文件并创建资源", func() {
			// 准备上传
			prepareReq := &service.PrepareUploadRequest{
				UserID:      userID,
				FileName:    fileInfo.Name(),
				FileSize:    fileInfo.Size(),
				ContentType: "text/plain",
				Ext:         "txt",
			}

			prepareResult, err := resourceService.PrepareUpload(ctx, prepareReq)
			So(err, ShouldBeNil)
			So(prepareResult, ShouldNotBeNil)
			So(prepareResult.SessionID, ShouldNotBeEmpty)

			// 上传文件内容到本地存储
			// 重置文件指针到开头
			_, err = novelFile.Seek(0, 0)
			So(err, ShouldBeNil)
			_, err = testStorage.Upload(ctx, prepareResult.UploadKey, novelFile, "text/plain")
			So(err, ShouldBeNil)

			// 完成上传
			completeReq := &service.CompleteUploadRequest{
				SessionID: prepareResult.SessionID,
			}

			completeResult, err := resourceService.CompleteUpload(ctx, completeReq)
			So(err, ShouldBeNil)
			So(completeResult, ShouldNotBeNil)
			So(completeResult.ResourceID, ShouldNotBeEmpty)

			Convey("步骤2: 创建小说", func() {
				novelID, err := novelService.CreateNovelFromResource(ctx, completeResult.ResourceID, userID, workflowID)
				So(err, ShouldBeNil)
				So(novelID, ShouldNotBeEmpty)

				Convey("步骤3: 切分章节", func() {
					targetChapters := 50
					err := novelService.SplitNovelIntoChapters(ctx, novelID, targetChapters)
					So(err, ShouldBeNil)

					Convey("验证结果", func() {
						// 验证小说存在
						novelEntity, err := novelRepo.FindByID(ctx, novelID)
						So(err, ShouldBeNil)
						So(novelEntity, ShouldNotBeNil)
						So(novelEntity.ResourceID, ShouldEqual, completeResult.ResourceID)

						// 验证章节已创建
						chapters, err := chapterRepo.FindByNovelID(ctx, novelID)
						So(err, ShouldBeNil)
						So(len(chapters), ShouldBeGreaterThan, 0)
						So(len(chapters), ShouldBeLessThanOrEqualTo, targetChapters+10) // 允许一些误差

						// 验证章节顺序和内容
						for i, ch := range chapters {
							So(ch.Sequence, ShouldEqual, i+1)
							So(ch.NovelID, ShouldEqual, novelID)
							So(ch.WorkflowID, ShouldEqual, workflowID)
							So(ch.UserID, ShouldEqual, userID)
							So(ch.Title, ShouldNotBeEmpty)
							So(ch.ChapterText, ShouldNotBeEmpty)
						}

						// 验证总字数
						totalChars := 0
						for _, ch := range chapters {
							totalChars += len([]rune(ch.ChapterText))
						}
						So(totalChars, ShouldBeGreaterThan, 0)
					})
				})
			})
		})
	})
}
