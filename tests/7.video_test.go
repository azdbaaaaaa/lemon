// Package tests 视频生成功能集成测试
//
// 运行测试：
//
//	MONGO_URI=mongodb://localhost:27017 ARK_API_KEY=your-key go test ./tests -run TestNovelService_GenerateVideos -v
//
// 说明：
//   - MONGO_URI: MongoDB 连接地址（默认: mongodb://localhost:27017）
//   - ARK_API_KEY: Ark API Key（必需，用于调用真实的 Ark 视频生成 API）
//   - ARK_VIDEO_MODEL: Ark 视频生成模型（可选，默认: doubao-seedance-1-0-lite-i2v-250428）
//   - FINISH_VIDEO_PATH: finish.mp4 文件路径（可选，默认: src/banner/finish_compatible.mp4）
//   - 测试会使用数据库中已有的图片和音频数据（需要先运行前面的测试生成图片和音频）
//   - 测试使用真实的 Ark Video Provider（调用真实 API）
//   - 测试完成后会自动清理测试数据库和临时存储文件
package tests

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lemon/internal/model/novel"
)

// TestNovelService_GenerateVideo_Narration 测试为章节生成所有 narration 视频
func TestNovelService_GenerateVideo_Narration(t *testing.T) {
	Convey("NovelService 生成 narration 视频测试", t, func() {
		// 使用 TestMain 中初始化的全局变量
		ctx := testCtx
		services := testServices

		userID := "test_user_novel_001"

		// 步骤1: 查找或创建测试章节（优先使用数据库中已有的）
		_, chapters := findOrCreateTestChapters(ctx, t, services, userID)
		So(len(chapters), ShouldBeGreaterThan, 0)

		chapterID := chapters[0].ID

		// 步骤2: 要求必须有测试解说文案、音频、字幕和图片，否则报错
		narrationID, _ := requireTestNarration(ctx, t, services, userID)
		So(narrationID, ShouldNotBeEmpty)

		// 要求必须有音频
		requireTestAudios(ctx, t, narrationID)

		// 要求必须有字幕
		requireTestSubtitles(ctx, t, narrationID)

		// 要求至少有2张图片
		requireTestImages(ctx, t, narrationID, 2)

		Convey("步骤3: 为章节生成所有 narration 视频", func() {
			// 为章节生成所有 narration 视频
			// 注意：这需要先有 first_video（前两张图片的视频）
			// 如果 first_video 还在处理中，narration 视频生成可能会失败
			// 这里先尝试生成，如果失败可以等待 first_video 完成后再重试

			videoIDs, err := services.NovelService.GenerateNarrationVideosForChapter(ctx, chapterID)
			if err != nil {
				// 如果失败，可能是因为 first_video 还在处理中
				// 等待一段时间后重试
				t.Logf("首次生成 narration 视频失败（可能 first_video 还在处理中），等待 10 秒后重试...")
				time.Sleep(10 * time.Second)
				videoIDs, err = services.NovelService.GenerateNarrationVideosForChapter(ctx, chapterID)
			}

			So(err, ShouldBeNil)
			So(len(videoIDs), ShouldBeGreaterThan, 0)

			Convey("验证 narration 视频已生成", func() {
				// 验证返回的 videoIDs 不为空
				So(len(videoIDs), ShouldBeGreaterThan, 0)

				// 可以进一步验证：
				// 1. 视频文件已上传到 resource 模块
				// 2. 视频记录已保存到数据库
				// 3. 每个 narration shot 对应一个视频（narration_01-03 合并成一个）
			})
		})
	})
}

// TestNovelService_GenerateVideo_Final 测试生成章节的最终完整视频
func TestNovelService_GenerateVideo_Final(t *testing.T) {
	Convey("NovelService 生成最终完整视频测试", t, func() {
		// 使用 TestMain 中初始化的全局变量
		ctx := testCtx
		services := testServices

		userID := "test_user_novel_001"

		// 步骤1: 查找或创建测试章节（优先使用数据库中已有的）
		_, chapters := findOrCreateTestChapters(ctx, t, services, userID)
		So(len(chapters), ShouldBeGreaterThan, 0)

		chapterID := chapters[0].ID

		// 步骤2: 要求必须有 narration 视频，否则报错
		requireTestNarrationVideos(ctx, t, chapterID)

		Convey("步骤3: 生成章节的最终完整视频（包含 finish.mp4）", func() {
			// 生成章节的最终完整视频
			videoID, err := services.NovelService.GenerateFinalVideoForChapter(ctx, chapterID)
			So(err, ShouldBeNil)
			So(videoID, ShouldNotBeEmpty)

			Convey("验证最终视频已生成", func() {
				// 验证返回的 videoID 不为空
				So(videoID, ShouldNotBeEmpty)

				// 可以进一步验证：
				// 1. 视频文件已上传到 resource 模块
				// 2. 视频记录已保存到数据库（video_type = "final_video"）
				// 3. 视频包含了所有 narration 视频和 finish.mp4
			})
		})
	})
}

// TestNovelService_GenerateVideos_CompleteFlow 测试完整的视频生成流程
// 这个测试会按顺序执行所有步骤，验证完整的视频生成流程
func TestNovelService_GenerateVideos_CompleteFlow(t *testing.T) {
	Convey("NovelService 完整视频生成流程测试", t, func() {
		// 使用 TestMain 中初始化的全局变量
		ctx := testCtx
		services := testServices

		userID := "test_user_novel_001"

		// 步骤1: 查找或创建测试章节
		_, chapters := findOrCreateTestChapters(ctx, t, services, userID)
		So(len(chapters), ShouldBeGreaterThan, 0)

		chapterID := chapters[0].ID

		// 步骤2: 要求必须有所有依赖（解说文案、音频、字幕、图片），否则报错
		narrationID, _ := requireTestNarration(ctx, t, services, userID)
		So(narrationID, ShouldNotBeEmpty)

		// 要求必须有音频
		requireTestAudios(ctx, t, narrationID)

		// 要求必须有字幕
		requireTestSubtitles(ctx, t, narrationID)

		// 要求至少有2张图片
		requireTestImages(ctx, t, narrationID, 2)

		Convey("步骤4: 生成所有 narration 视频", func() {
			videoIDs, err := services.NovelService.GenerateNarrationVideosForChapter(ctx, chapterID)
			So(err, ShouldBeNil)
			So(len(videoIDs), ShouldBeGreaterThan, 0)

			// 等待 narration 视频完成
			waitForVideosComplete(ctx, t, services, chapterID, "narration_video", 120*time.Second)
		})

		Convey("步骤5: 生成最终完整视频（包含 finish.mp4）", func() {
			videoID, err := services.NovelService.GenerateFinalVideoForChapter(ctx, chapterID)
			So(err, ShouldBeNil)
			So(videoID, ShouldNotBeEmpty)
		})
	})
}

// waitForVideosComplete 等待视频生成完成
// 轮询检查视频状态，直到所有视频都完成或超时
func waitForVideosComplete(ctx context.Context, t *testing.T, services *TestServices, chapterID, videoType string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("上下文已取消")
		case <-ticker.C:
			// 直接查询数据库获取视频状态
			var videoModel novel.ChapterVideo
			videoColl := testDB.Collection(videoModel.Collection())
			videoFilter := bson.M{"chapter_id": chapterID, "video_type": videoType, "deleted_at": nil}
			cursor, err := videoColl.Find(ctx, videoFilter, options.Find().SetSort(bson.M{"sequence": 1}))
			if err != nil {
				t.Logf("查询视频状态失败: %v", err)
				continue
			}
			defer cursor.Close(ctx)

			var videos []*novel.ChapterVideo
			if err := cursor.All(ctx, &videos); err != nil {
				t.Logf("解析视频数据失败: %v", err)
				continue
			}

			if len(videos) == 0 {
				t.Logf("还没有生成任何 %s 视频，继续等待...", videoType)
				continue
			}

			// 检查所有视频是否都已完成
			allCompleted := true
			hasFailed := false
			for _, video := range videos {
				if video.Status == "failed" {
					hasFailed = true
					t.Logf("视频 %s 生成失败: %s", video.ID, video.ErrorMessage)
				} else if video.Status != "completed" {
					allCompleted = false
					t.Logf("视频 %s 状态: %s", video.ID, video.Status)
				}
			}

			if allCompleted {
				t.Logf("所有 %s 视频已完成", videoType)
				return
			}

			if hasFailed {
				t.Fatalf("部分 %s 视频生成失败", videoType)
			}

			if time.Now().After(deadline) {
				t.Fatalf("等待 %s 视频完成超时（%v）", videoType, timeout)
			}

			t.Logf("等待 %s 视频完成...（已等待 %v）", videoType, time.Since(deadline.Add(-timeout)))
		}
	}
}
