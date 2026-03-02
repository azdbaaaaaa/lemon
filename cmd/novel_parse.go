package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var novelParseCmd = &cobra.Command{
	Use:   "parse",
	Short: "解析小说并切分章节",
	Long: `根据已有的小说ID，读取其原始资源文本并切分章节。

示例：
  lemon novel parse --novel-id nvl_abc123
  lemon novel parse --novel-id nvl_abc123 --chapters 30`,
	RunE: runNovelParse,
}

func init() {
	parseFlags := novelParseCmd.Flags()
	parseFlags.String("novel-id", "", "要解析的小说ID")
	parseFlags.StringP("user", "u", "", "（可选）小说所属用户ID，默认使用小说记录中的用户ID")
	parseFlags.Int("chapters", 50, "目标章节数（内部最多切分为10章）")
	_ = novelParseCmd.MarkFlagRequired("novel-id")
}

// runNovelParse 执行解析小说并切分章节命令
func runNovelParse(cmd *cobra.Command, args []string) error {
	env, err := newNovelCLIEnv()
	if err != nil {
		return err
	}
	defer env.Close()

	novelID, err := cmd.Flags().GetString("novel-id")
	if err != nil {
		return err
	}
	targetChapters, err := cmd.Flags().GetInt("chapters")
	if err != nil {
		return err
	}

	if novelID == "" {
		return fmt.Errorf("必须指定 --novel-id 参数")
	}
	if targetChapters <= 0 {
		targetChapters = 50
	}

	// 先确认小说存在，便于提前给出更清晰的错误信息
	if _, err := env.novelSvc.GetNovel(env.baseCtx, novelID); err != nil {
		return fmt.Errorf("获取小说失败: %w", err)
	}

	if err := env.novelSvc.SplitChapters(env.baseCtx, novelID, targetChapters); err != nil {
		return fmt.Errorf("切分章节失败: %w", err)
	}

	chapters, err := env.novelSvc.GetChapters(env.baseCtx, novelID)
	if err != nil {
		return fmt.Errorf("获取章节列表失败: %w", err)
	}

	fmt.Printf("解析完成！\n  NovelID: %s\n  Chapters: %d\n", novelID, len(chapters))
	return nil
}
