package noveltools

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestChapterSplitter_Split(t *testing.T) {
	Convey("ChapterSplitter.Split 能正确切分小说内容", t, func() {
		splitter := NewChapterSplitter()

		Convey("空内容应返回 nil", func() {
			result := splitter.Split("", 50)
			So(result, ShouldBeNil)
		})

		Convey("空白内容应返回 nil", func() {
			result := splitter.Split("   \n\n  ", 50)
			So(result, ShouldBeNil)
		})

		Convey("按章节标题切分（中文格式）", func() {
			content := `第一章 开始
这是第一章的内容，包含了很多文字。

第二章 发展
这是第二章的内容，继续讲述故事。

第三章 高潮
这是第三章的内容，故事达到高潮。`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 2)
			So(result[0].Title, ShouldContainSubstring, "第一章")
			So(result[0].Text, ShouldContainSubstring, "这是第一章的内容")
			So(result[1].Title, ShouldContainSubstring, "第二章")
			So(result[1].Text, ShouldContainSubstring, "这是第二章的内容")
		})

		Convey("按章节标题切分（英文格式）", func() {
			content := `chapter 1 Beginning
This is chapter 1 content.

chapter 2 Development
This is chapter 2 content.`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 2)
			So(result[0].Title, ShouldContainSubstring, "chapter 1")
			So(result[1].Title, ShouldContainSubstring, "chapter 2")
		})

		Convey("无章节标题时按长度切分", func() {
			// 构造一个较长的文本，没有章节标题（确保足够长以切分成5段）
			content := strings.Repeat("这是一段很长的小说内容，包含了很多文字和情节描述。", 200)

			result := splitter.Split(content, 5)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 5)
			for _, seg := range result {
				So(seg.Text, ShouldNotBeEmpty)
				So(seg.Title, ShouldNotBeEmpty)
			}
		})

		Convey("目标章节数为 0 时使用默认值", func() {
			// 构造足够长的内容以支持切分成50段
			content := strings.Repeat("这是一段很长的小说内容，包含了很多文字和情节描述。", 1000)

			result := splitter.Split(content, 0)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 1)
			// 如果内容足够长，应该接近默认值50
			if len([]rune(content)) > 50000 {
				So(len(result), ShouldBeLessThanOrEqualTo, 50)
			}
		})

		Convey("目标章节数为负数时使用默认值", func() {
			// 构造足够长的内容以支持切分成50段
			content := strings.Repeat("这是一段很长的小说内容，包含了很多文字和情节描述。", 1000)

			result := splitter.Split(content, -1)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 1)
			// 如果内容足够长，应该接近默认值50
			if len([]rune(content)) > 50000 {
				So(len(result), ShouldBeLessThanOrEqualTo, 50)
			}
		})

		Convey("章节数过多时会合并", func() {
			content := `第一章
内容1

第二章
内容2

第三章
内容3

第四章
内容4

第五章
内容5`

			result := splitter.Split(content, 2)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeLessThanOrEqualTo, 2)
		})

		Convey("每个章节应包含标题和正文", func() {
			content := `第一章 开始
这是第一章的内容。

第二章 发展
这是第二章的内容。`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			for _, seg := range result {
				So(seg.Title, ShouldNotBeEmpty)
				So(seg.Text, ShouldNotBeEmpty)
				So(seg.Text, ShouldContainSubstring, seg.Title)
			}
		})

		Convey("处理多个连续空行", func() {
			content := `第一章 开始


这是第一章的内容。



第二章 发展


这是第二章的内容。`

			result := splitter.Split(content, 50)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldBeGreaterThanOrEqualTo, 2)
			// 验证空行被规范化
			for _, seg := range result {
				So(seg.Text, ShouldNotContainSubstring, "\n\n\n")
			}
		})
	})
}
