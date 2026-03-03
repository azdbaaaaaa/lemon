package novel

import (
	"github.com/spf13/cobra"
)

// novelCmd 顶层小说命令
var novelCmd = &cobra.Command{
	Use:   "novel",
	Short: "小说相关工具命令",
	Long:  "提供上传小说文件、角色资产生成等命令。",
}

// NewNovelCommand 创建并返回小说根命令
func NewNovelCommand() *cobra.Command {
	return novelCmd
}
