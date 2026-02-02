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

		Convey("步骤1: 使用服务端上传方式上传文件", func() {
			// 重置文件指针到开头
			_, err = novelFile.Seek(0, 0)
			So(err, ShouldBeNil)

			// 使用服务端上传方式（UploadFile）
			uploadReq := &service.UploadFileRequest{
				UserID:      userID,
				FileName:    fileStat.Name(),
				ContentType: "text/plain",
				Ext:         "txt",
				Data:        novelFile,
			}

			uploadResult, err := services.ResourceService.UploadFile(ctx, uploadReq)
			So(err, ShouldBeNil)
			So(uploadResult.ResourceID, ShouldNotBeEmpty)
			So(uploadResult.FileSize, ShouldEqual, fileStat.Size())
			Convey("步骤2: 验证资源记录", func() {
				// 查询资源详情
				resourceResult, err := services.ResourceService.GetResource(ctx, &service.GetResourceRequest{
					ResourceID: uploadResult.ResourceID,
					UserID:     userID,
				})
				So(err, ShouldBeNil)
				resourceEntity := resourceResult.Resource
				So(resourceEntity.UserID, ShouldEqual, userID)
				So(resourceEntity.Name, ShouldEqual, fileStat.Name())
				So(resourceEntity.Ext, ShouldEqual, "txt")
				So(resourceEntity.FileSize, ShouldEqual, fileStat.Size())
				So(resourceEntity.ContentType, ShouldEqual, "text/plain")
				So(resourceEntity.StorageKey, ShouldNotBeEmpty)
				So(resourceEntity.StorageType, ShouldEqual, "local")
				So(string(resourceEntity.Status), ShouldEqual, "ready")

				Convey("步骤3: 验证可以下载文件", func() {
					// 获取下载URL
					downloadReq := &service.GetDownloadURLRequest{
						UserID:     userID,
						ResourceID: uploadResult.ResourceID,
						ExpiresIn:  time.Hour,
					}
					downloadResult, err := services.ResourceService.GetDownloadURL(ctx, downloadReq)
					So(err, ShouldBeNil)
					So(downloadResult.DownloadURL, ShouldNotBeEmpty)

					Convey("步骤4: 验证可以下载文件内容", func() {
						// 使用 DownloadFile 下载文件
						downloadFileReq := &service.DownloadFileRequest{
							UserID:     userID,
							ResourceID: uploadResult.ResourceID,
						}
						downloadFileResult, err := services.ResourceService.DownloadFile(ctx, downloadFileReq)
						So(err, ShouldBeNil)
						defer downloadFileResult.Data.Close()

						// 读取文件内容的前几个字节进行验证
						buffer := make([]byte, 100)
						n, err := downloadFileResult.Data.Read(buffer)
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
