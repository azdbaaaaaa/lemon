package cmd

import (
	novelcmd "lemon/cmd/novel"
)

// 将 novel 相关子命令挂载到 rootCmd 下
func init() {
	rootCmd.AddCommand(novelcmd.NewNovelCommand())
}
