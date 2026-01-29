// Package tests 集成测试
//
// 运行集成测试：
//
//	INTEGRATION_TEST=true MONGO_URI=mongodb://localhost:27017 go test ./tests -run TestNovelService_Integration -v
//
// 说明：
//   - INTEGRATION_TEST=true: 启用集成测试（默认跳过）
//   - MONGO_URI: MongoDB 连接地址（默认: mongodb://localhost:27017）
//   - 测试会读取 assets/novel/001/大道主(飘荡的云).txt 作为测试数据
//   - 测试使用内存存储（memoryStorage）模拟本地存储
//   - 测试完成后会自动清理测试数据库
package tests

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/pkg/id"
	"lemon/internal/pkg/storage"
	novelrepo "lemon/internal/repository/novel"
	resourceRepo "lemon/internal/repository/resource"
	"lemon/internal/service"
)

// memoryStorage 内存存储实现（用于测试）
type memoryStorage struct {
	files map[string][]byte
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{
		files: make(map[string][]byte),
	}
}

func (m *memoryStorage) Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
	content, err := io.ReadAll(data)
	if err != nil {
		return "", err
	}
	m.files[key] = content
	return key, nil
}

func (m *memoryStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	content, ok := m.files[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &memoryReader{data: content, pos: 0}, nil
}

func (m *memoryStorage) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (string, error) {
	return "memory://" + key, nil
}

func (m *memoryStorage) GetPresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	return "memory://" + key, nil
}

func (m *memoryStorage) Delete(ctx context.Context, key string) error {
	delete(m.files, key)
	return nil
}

func (m *memoryStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.files[key]
	return ok, nil
}

func (m *memoryStorage) GetFileInfo(ctx context.Context, key string) (*storage.FileInfo, error) {
	content, ok := m.files[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &storage.FileInfo{
		Key:          key,
		Size:         int64(len(content)),
		ContentType:  "text/plain",
		LastModified: time.Now(),
	}, nil
}

func (m *memoryStorage) GetStorageType() string {
	return "memory"
}

type memoryReader struct {
	data []byte
	pos  int
}

func (r *memoryReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *memoryReader) Close() error {
	return nil
}

func TestNovelService_Integration(t *testing.T) {
	Convey("NovelService 集成测试：完整的小说处理流程", t, func() {
		// 跳过测试，除非明确设置环境变量
		if os.Getenv("INTEGRATION_TEST") != "true" {
			t.Skip("跳过集成测试（设置 INTEGRATION_TEST=true 以运行）")
		}

		ctx := context.Background()

		// 1. 初始化 MongoDB 连接（使用测试数据库）
		mongoURI := os.Getenv("MONGO_URI")
		if mongoURI == "" {
			mongoURI = "mongodb://localhost:27017"
		}

		mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		So(err, ShouldBeNil)
		defer func() {
			_ = mongoClient.Disconnect(ctx)
		}()

		// 使用测试数据库
		db := mongoClient.Database("lemon_test")

		// 清理测试数据
		defer func() {
			_ = db.Collection("resources").Drop(ctx)
			_ = db.Collection("upload_sessions").Drop(ctx)
			_ = db.Collection("novels").Drop(ctx)
			_ = db.Collection("chapters").Drop(ctx)
		}()

		// 2. 初始化存储（内存存储）
		memStorage := newMemoryStorage()

		// 3. 初始化仓库
		resourceRepo := resourceRepo.NewResourceRepo(db)
		novelRepo := novelrepo.NewNovelRepo(db)
		chapterRepo := novelrepo.NewChapterRepo(db)

		// 4. 初始化服务
		resourceService := service.NewResourceService(resourceRepo, memStorage)
		novelService := service.NewNovelService(resourceRepo, novelRepo, chapterRepo, memStorage, nil)

		// 5. 读取测试文件
		// 从项目根目录查找文件（tests 目录在项目根目录下）
		novelFilePath := filepath.Join("assets", "novel", "001", "大道主(飘荡的云).txt")
		if _, err := os.Stat(novelFilePath); os.IsNotExist(err) {
			// 如果从当前目录找不到，尝试从测试文件所在目录向上查找
			// 测试文件在 tests/，需要向上一级到项目根目录
			novelFilePath = filepath.Join("..", "assets", "novel", "001", "大道主(飘荡的云).txt")
		}
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

			// 上传文件内容到内存存储
			// 重置文件指针到开头
			_, err = novelFile.Seek(0, 0)
			So(err, ShouldBeNil)
			_, err = memStorage.Upload(ctx, prepareResult.UploadKey, novelFile, "text/plain")
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
