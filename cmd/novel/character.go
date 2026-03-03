package novel

import (
	"fmt"

	"github.com/spf13/cobra"
)

// characterCmd 基于小说内容生成角色资产
var characterCmd = &cobra.Command{
	Use:   "character",
	Short: "从小说内容生成角色资产（角色与重要道具）",
	Long: `基于已切分章节的整部小说内容，调用 NovelService 生成角色和道具资产。

示例：
  lemon novel character --novel-id nvl_xxx`,
	RunE: runCharacter,
}

func init() {
	novelCmd.AddCommand(characterCmd)

	flags := characterCmd.Flags()
	flags.String("novel-id", "", "要处理的小说ID")
	_ = characterCmd.MarkFlagRequired("novel-id")
}

// runCharacter 执行角色资产生成命令
func runCharacter(cmd *cobra.Command, args []string) error {
	env, err := newCLIEnv()
	if err != nil {
		return err
	}
	defer env.Close()

	novelID, err := cmd.Flags().GetString("novel-id")
	if err != nil {
		return err
	}
	if novelID == "" {
		return fmt.Errorf("必须指定 --novel-id 参数")
	}

	if err := env.novelSvc.GenerateCharactersFromNovel(env.baseCtx, novelID); err != nil {
		return fmt.Errorf("生成角色资产失败: %w", err)
	}

	fmt.Printf("从小说 %s 生成角色与道具资产成功\n", novelID)
	return nil
}

