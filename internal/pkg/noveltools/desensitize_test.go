package noveltools

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestContentFilter_CheckContent(t *testing.T) {
	Convey("CheckContent 能识别违禁/严重违禁词", t, func() {
		filter := NewContentFilter()

		Convey("正常内容应通过", func() {
			result := filter.CheckContent("这是一段正常的内容，没有任何敏感词汇。")
			So(result, ShouldNotBeNil)
			So(result.IsSafe, ShouldBeTrue)
			So(result.Issues, ShouldBeEmpty)
		})

		Convey("包含违禁词汇应不通过", func() {
			result := filter.CheckContent("这里提到了毒品相关的内容。")
			So(result.IsSafe, ShouldBeFalse)
			So(strings.Join(result.Issues, ","), ShouldContainSubstring, "毒品")
		})

		Convey("包含严重违禁词汇应不通过", func() {
			result := filter.CheckContent("这里有一些色情内容。")
			So(result.IsSafe, ShouldBeFalse)
			So(strings.Join(result.Issues, ","), ShouldContainSubstring, "色情")
		})

		Convey("混合内容包含强暴应不通过", func() {
			result := filter.CheckContent("正常内容，但是提到了强暴这个词。")
			So(result.IsSafe, ShouldBeFalse)
			So(strings.Join(result.Issues, ","), ShouldContainSubstring, "强暴")
		})
	})
}

func TestContentFilter_FilterContent(t *testing.T) {
	Convey("FilterContent 能替换/移除敏感词", t, func() {
		filter := NewContentFilter()

		Convey("替换敏感词", func() {
			got := filter.FilterContent("罪犯被警察抓到了监狱。")
			So(got, ShouldEqual, "嫌疑人被jc抓到了牢狱。")
			So(got, ShouldNotContainSubstring, "罪犯")
			So(got, ShouldNotContainSubstring, "警察")
			So(got, ShouldNotContainSubstring, "监狱")
		})

		Convey("移除严重违禁词", func() {
			got := filter.FilterContent("这里有一些色情和毒品相关的内容。")
			So(got, ShouldEqual, "这里有一些和相关的内容。")
			So(got, ShouldNotContainSubstring, "色情")
			So(got, ShouldNotContainSubstring, "毒品")
		})

		Convey("替换和移除混合", func() {
			got := filter.FilterContent("罪犯睡觉时提到了色情内容。")
			So(got, ShouldEqual, "嫌疑人休息时提到了内容。")
			So(got, ShouldNotContainSubstring, "罪犯")
			So(got, ShouldNotContainSubstring, "睡觉")
			So(got, ShouldNotContainSubstring, "色情")
		})

		Convey("词汇替换", func() {
			got := filter.FilterContent("他温柔地拥抱了她。")
			So(got, ShouldEqual, "他和善地相伴了她。")
			So(got, ShouldNotContainSubstring, "温柔")
			So(got, ShouldNotContainSubstring, "拥抱")
		})
	})
}

func TestContentFilter_ProcessContent(t *testing.T) {
	Convey("ProcessContent 能同时检查和过滤", t, func() {
		filter := NewContentFilter()
		originalContent := "这里有毒品和强暴的描述，罪犯被警察抓进监狱，床上有很多色情物品。"
		expectedFilteredContent := "这里有和的描述，嫌疑人被jc抓进牢狱，有很多物品。"

		filteredContent, checkResult := filter.ProcessContent(originalContent)
		So(filteredContent, ShouldEqual, expectedFilteredContent)
		So(checkResult.IsSafe, ShouldBeFalse)
		So(len(checkResult.Issues), ShouldBeGreaterThan, 0)

		// 验证过滤后的内容应该安全
		checkResultAfterFilter := filter.CheckContent(filteredContent)
		So(checkResultAfterFilter.IsSafe, ShouldBeTrue)
	})
}
