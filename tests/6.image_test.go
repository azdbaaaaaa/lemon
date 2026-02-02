// Package tests 图片生成功能集成测试
//
// 运行测试：
//
//	MONGO_URI=mongodb://localhost:27017 go test ./tests -run TestNovelService_GenerateImages -v
//
// 说明：
//   - MONGO_URI: MongoDB 连接地址（默认: mongodb://localhost:27017）
//   - 图片生成提供者在 TestMain 中初始化（使用 T2P Provider）
//   - T2P 配置：VOLCENGINE_ACCESS_KEY, VOLCENGINE_SECRET_KEY
//   - 测试会使用数据库中已有的解说文案数据（需要先运行 3.narration_test.go）
//   - 测试完成后会自动清理测试数据库和临时存储文件
package tests

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TestNovelService_GenerateImages 测试根据解说文案生成场景特写图片
func TestNovelService_GenerateImages(t *testing.T) {
	Convey("NovelService 生成图片测试", t, func() {
		// 使用 TestMain 中初始化的全局变量
		ctx := testCtx
		services := testServices

		userID := "test_user_novel_001"

		// 步骤1: 要求必须有测试解说文案，否则报错
		narrationID, _ := requireTestNarration(ctx, t, services, userID)
		So(narrationID, ShouldNotBeEmpty)

		Convey("步骤2: 为解说文案生成场景特写图片", func() {
			// 为解说文案生成所有场景特写图片
			imageIDs, err := services.NovelService.GenerateImagesForNarration(ctx, narrationID)
			So(err, ShouldBeNil)
			So(len(imageIDs), ShouldBeGreaterThan, 0)

			Convey("验证图片已保存到 chapter_images 表", func() {
				// 验证返回的 imageIDs 不为空
				So(len(imageIDs), ShouldBeGreaterThan, 0)

				// 可以进一步验证：
				// 1. 图片文件已上传到 resource 模块
				// 2. 图片记录已保存到数据库
				// 3. 每个 NarrationShot 对应一张图片
				// 注意：目前 NovelService 没有 GetChapterImage 方法，所以无法直接验证数据库记录
				// 如果需要验证，可以添加 GetChapterImage 方法到 NovelService 接口
			})
		})
	})
}
