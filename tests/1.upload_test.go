// Package tests 上传功能集成测试
//
// 运行上传测试：
//
//	MONGO_URI=mongodb://localhost:27017 go test ./tests -run TestResourceService_UploadTXT -v
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
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"lemon/internal/service"
)

// TestResourceService_UploadTXT 测试上传 TXT 文件的完整流程
func TestResourceService_UploadTXT(t *testing.T) {
	Convey("ResourceService 上传 TXT 文件测试", t, func() {
		// 使用 TestMain 中初始化的全局变量
		ctx := testCtx
		services := testServices

		// 读取测试文件
		novelFilePath := getTestNovelFilePath(t)
		novelFile, err := os.Open(novelFilePath)
		if err != nil {
			t.Fatalf("应该能打开测试文件: %s, 错误: %v", novelFilePath, err)
		}
		defer novelFile.Close()

		fileStat, err := novelFile.Stat()
		So(err, ShouldBeNil)

		userID := "test_user_upload_001"

		Convey("步骤1: 上传文件到存储", func() {
			// 准备上传（创建上传会话，但不验证返回值）
			prepareReq := &service.PrepareUploadRequest{
				UserID:      userID,
				FileName:    fileStat.Name(),
				FileSize:    fileStat.Size(),
				ContentType: "text/plain",
				Ext:         "txt",
			}

			prepareResult, err := services.ResourceService.PrepareUpload(ctx, prepareReq)
			So(err, ShouldBeNil)
			So(prepareResult, ShouldNotBeNil)
			sessionID := prepareResult.SessionID
			uploadKey := prepareResult.UploadKey

			// 重置文件指针到开头
			_, err = novelFile.Seek(0, 0)
			So(err, ShouldBeNil)

			// 上传文件
			uploadURL, err := services.Storage.Upload(ctx, uploadKey, novelFile, "text/plain")
			So(err, ShouldBeNil)
			So(uploadURL, ShouldNotBeEmpty)

			// 验证文件已上传
			exists, err := services.Storage.Exists(ctx, uploadKey)
			So(err, ShouldBeNil)
			So(exists, ShouldBeTrue)

			Convey("步骤2: 完成上传（创建资源记录）", func() {
				completeReq := &service.CompleteUploadRequest{
					SessionID: sessionID,
					// MD5 和 SHA256 可选，这里不提供
				}

				completeResult, err := services.ResourceService.CompleteUpload(ctx, completeReq)
				So(err, ShouldBeNil)
				So(completeResult, ShouldNotBeNil)
				So(completeResult.ResourceID, ShouldNotBeEmpty)
				So(completeResult.FileSize, ShouldEqual, fileStat.Size())

				Convey("步骤3: 验证资源记录", func() {
					// 查询资源详情
					resourceEntity, err := services.ResourceRepo.FindByID(ctx, completeResult.ResourceID)
					So(err, ShouldBeNil)
					So(resourceEntity, ShouldNotBeNil)
					So(resourceEntity.UserID, ShouldEqual, userID)
					So(resourceEntity.Name, ShouldEqual, fileStat.Name())
					So(resourceEntity.Ext, ShouldEqual, "txt")
					So(resourceEntity.FileSize, ShouldEqual, fileStat.Size())
					So(resourceEntity.ContentType, ShouldEqual, "text/plain")
					So(resourceEntity.StorageKey, ShouldEqual, uploadKey)
					So(resourceEntity.StorageType, ShouldEqual, "local")
					So(string(resourceEntity.Status), ShouldEqual, "ready")

					Convey("步骤4: 验证可以下载文件", func() {
						// 获取下载URL
						downloadReq := &service.GetDownloadURLRequest{
							ResourceID: completeResult.ResourceID,
							ExpiresIn:  time.Hour,
						}
						downloadResult, err := services.ResourceService.GetDownloadURL(ctx, userID, downloadReq)
						So(err, ShouldBeNil)
						So(downloadResult, ShouldNotBeNil)
						So(downloadResult.DownloadURL, ShouldNotBeEmpty)

						// 下载文件并验证内容
						reader, err := services.Storage.Download(ctx, uploadKey)
						So(err, ShouldBeNil)
						defer reader.Close()

						// 读取文件内容的前几个字节进行验证
						buffer := make([]byte, 100)
						n, err := reader.Read(buffer)
						So(err, ShouldBeNil)
						So(n, ShouldBeGreaterThan, 0)

						// 验证文件内容不为空
						_, err = novelFile.Seek(0, 0)
						So(err, ShouldBeNil)
						expectedBuffer := make([]byte, 100)
						expectedN, err := novelFile.Read(expectedBuffer)
						So(err, ShouldBeNil)
						So(n, ShouldEqual, expectedN)
						So(string(buffer[:n]), ShouldEqual, string(expectedBuffer[:expectedN]))
					})
				})
			})
		})
	})
}
