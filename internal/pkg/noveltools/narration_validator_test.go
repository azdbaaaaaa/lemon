package noveltools

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNarrationValidator_Validate(t *testing.T) {
	Convey("NarrationValidator.Validate 能正确验证解说内容", t, func() {
		validator := NewNarrationValidator()

		Convey("空内容应返回无效", func() {
			result := validator.Validate("", 1100, 1300, 0)
			So(result.IsValid, ShouldBeFalse)
			So(result.Message, ShouldEqual, "解说内容为空")
		})

		Convey("空白内容应返回无效", func() {
			result := validator.Validate("   \n\n  ", 1100, 1300, 0)
			So(result.IsValid, ShouldBeFalse)
			So(result.Message, ShouldEqual, "解说内容为空")
		})

		Convey("未找到解说内容标签应返回无效", func() {
			content := "<分镜1>这是分镜内容</分镜1>"
			result := validator.Validate(content, 1100, 1300, 0)
			So(result.IsValid, ShouldBeFalse)
			So(result.Message, ShouldEqual, "未找到解说内容标签")
		})

		Convey("正常解说内容应通过验证", func() {
			content := `<分镜1>
<解说内容>这是一段足够长的解说内容，用于测试验证功能。` + strings.Repeat("文字", 300) + `</解说内容>
</分镜1>`

			result := validator.Validate(content, 1100, 1300, 0)
			So(result.IsValid, ShouldBeTrue)
			So(result.Message, ShouldNotBeEmpty)
		})

		Convey("字数不足应产生警告", func() {
			content := `<分镜1>
<解说内容>短内容</解说内容>
</分镜1>`

			result := validator.Validate(content, 1100, 1300, 0)
			So(result.IsValid, ShouldBeTrue)
			So(len(result.Warnings), ShouldBeGreaterThan, 0)
			So(strings.Join(result.Warnings, ""), ShouldContainSubstring, "长度不足")
		})

		Convey("字数过长应产生警告", func() {
			content := `<分镜1>
<解说内容>` + strings.Repeat("这是一段很长的解说内容。", 200) + `</解说内容>
</分镜1>`

			result := validator.Validate(content, 1100, 1300, 0)
			So(result.IsValid, ShouldBeTrue)
			So(len(result.Warnings), ShouldBeGreaterThan, 0)
			So(strings.Join(result.Warnings, ""), ShouldContainSubstring, "过长")
		})

		Convey("解说内容数量不匹配应产生警告", func() {
			content := `<分镜1>
<解说内容>内容1</解说内容>
</分镜1>
<分镜2>
<解说内容>内容2</解说内容>
</分镜2>`

			result := validator.Validate(content, 1100, 1300, 21)
			So(result.IsValid, ShouldBeTrue)
			So(len(result.Warnings), ShouldBeGreaterThan, 0)
			So(strings.Join(result.Warnings, ""), ShouldContainSubstring, "数量不正确")
		})

		Convey("应自动移除不需要的标签", func() {
			content := `<分镜1>
<角色编号>001</角色编号>
<解说内容>这是解说内容</解说内容>
<风格>现代</风格>
</分镜1>`

			result := validator.Validate(content, 1, 10000, 0)
			So(result.IsValid, ShouldBeTrue)
			So(result.Message, ShouldNotContainSubstring, "<角色编号>")
			So(result.Message, ShouldNotContainSubstring, "<风格>")
		})

		Convey("应自动修复未闭合的XML标签", func() {
			content := `<分镜1>
<解说内容>这是解说内容</解说内容>
<分镜1>`

			result := validator.Validate(content, 1, 10000, 0)
			So(result.IsValid, ShouldBeTrue)
			So(result.Message, ShouldContainSubstring, "</分镜1>")
		})

		Convey("检测到敏感内容应产生警告但不阻止", func() {
			content := `<分镜1>
<解说内容>这里提到了毒品相关的内容。` + strings.Repeat("文字", 300) + `</解说内容>
</分镜1>`

			result := validator.Validate(content, 1100, 1300, 0)
			So(result.IsValid, ShouldBeTrue)
			// 应该包含敏感内容警告
			hasWarning := false
			for _, warning := range result.Warnings {
				if strings.Contains(warning, "敏感内容") {
					hasWarning = true
					break
				}
			}
			So(hasWarning, ShouldBeTrue)
		})
	})
}

func TestNarrationValidator_removeUnwantedTags(t *testing.T) {
	Convey("removeUnwantedTags 能正确移除不需要的标签", t, func() {
		validator := NewNarrationValidator()

		Convey("应移除指定的标签", func() {
			content := `<分镜1>
<角色编号>001</角色编号>
<解说内容>内容</解说内容>
<风格>现代</风格>
</分镜1>`

			result := validator.removeUnwantedTags(content)
			So(result, ShouldNotContainSubstring, "<角色编号>")
			So(result, ShouldNotContainSubstring, "</角色编号>")
			So(result, ShouldNotContainSubstring, "<风格>")
			So(result, ShouldNotContainSubstring, "</风格>")
			So(result, ShouldContainSubstring, "<解说内容>")
		})

		Convey("应移除单独的标签", func() {
			content := `<分镜1>
<角色类型>
<解说内容>内容</解说内容>
</分镜1>`

			result := validator.removeUnwantedTags(content)
			So(result, ShouldNotContainSubstring, "<角色类型>")
		})

		Convey("应清理多余的空行", func() {
			content := `<分镜1>


<解说内容>内容</解说内容>


</分镜1>`

			result := validator.removeUnwantedTags(content)
			So(result, ShouldNotContainSubstring, "\n\n\n")
		})
	})
}

func TestNarrationValidator_fixXMLTags(t *testing.T) {
	Convey("fixXMLTags 能正确修复XML标签闭合", t, func() {
		validator := NewNarrationValidator()

		Convey("应修复未闭合的标签", func() {
			content := `<分镜1>
<解说内容>内容</解说内容>`

			result := validator.fixXMLTags(content)
			So(result, ShouldContainSubstring, "</分镜1>")
		})

		Convey("应修复多个未闭合的标签", func() {
			content := `<分镜1>
<分镜2>
<解说内容>内容</解说内容>`

			result := validator.fixXMLTags(content)
			So(result, ShouldContainSubstring, "</分镜2>")
			So(result, ShouldContainSubstring, "</分镜1>")
		})

		Convey("已闭合的标签不应重复添加", func() {
			content := `<分镜1>
<解说内容>内容</解说内容>
</分镜1>`

			result := validator.fixXMLTags(content)
			// 统计 </分镜1> 出现次数
			count := strings.Count(result, "</分镜1>")
			So(count, ShouldEqual, 1)
		})
	})
}
