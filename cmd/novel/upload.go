package novel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"lemon/internal/model/novel"
	"lemon/internal/service"
)

// uploadCmd 上传小说文件并创建资源 + 小说 + 章节
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "上传小说文本文件并创建资源记录",
	Long: `上传小说文本文件到存储系统并创建资源记录，并自动创建小说和章节。

示例：
  lemon novel upload --file assets/novel/001/xxx.txt --user user123
  lemon novel upload -f ./novel.txt -u user123 --ext txt`,
	RunE: runUpload,
}

func init() {
	novelCmd.AddCommand(uploadCmd)

	flags := uploadCmd.Flags()
	flags.StringP("file", "f", "", "小说文本文件路径（例如：assets/novel/001/xxx.txt）")
	flags.StringP("user", "u", "", "资源所属用户ID")
	flags.String("content-type", "text/plain", "文件内容类型（MIME type）")
	flags.Int("chapters", 50, "目标章节数（内部最多切分为10章）")
	flags.String("ext", "", "文件扩展名（不含点号，默认根据文件名推断）")

	_ = uploadCmd.MarkFlagRequired("file")
	_ = uploadCmd.MarkFlagRequired("user")
}

// runUpload 执行上传小说文件命令
func runUpload(cmd *cobra.Command, args []string) error {
	env, err := newCLIEnv()
	if err != nil {
		return err
	}
	defer env.Close()

	filePath, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	userID, err := cmd.Flags().GetString("user")
	if err != nil {
		return err
	}
	contentType, err := cmd.Flags().GetString("content-type")
	if err != nil {
		return err
	}
	targetChapters, err := cmd.Flags().GetInt("chapters")
	if err != nil {
		return err
	}
	extFlag, err := cmd.Flags().GetString("ext")
	if err != nil {
		return err
	}

	if filePath == "" {
		return fmt.Errorf("必须指定 --file 参数")
	}
	if userID == "" {
		return fmt.Errorf("必须指定 --user 参数")
	}
	if targetChapters <= 0 {
		targetChapters = 50
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	ext := extFlag
	if ext == "" {
		ext = strings.TrimPrefix(filepath.Ext(stat.Name()), ".")
	}
	if ext == "" {
		ext = "txt"
	}

	req := &service.UploadFileRequest{
		UserID:      userID,
		FileName:    stat.Name(),
		ContentType: contentType,
		Ext:         ext,
		Data:        file,
	}

	result, err := env.resourceSvc.UploadFile(env.baseCtx, req)
	if err != nil {
		return fmt.Errorf("上传小说文件失败: %w", err)
	}

	// 根据资源创建小说（使用默认的解说类型和风格）
	novelID, err := env.novelSvc.CreateNovelFromResource(
		env.baseCtx,
		result.ResourceID,
		userID,
		novel.NarrationTypeNarration,
		novel.NovelStyleAnime,
	)
	if err != nil {
		return fmt.Errorf("根据资源创建小说失败: %w", err)
	}

	// 上传完成后自动切分章节
	if err := env.novelSvc.SplitChapters(env.baseCtx, novelID, targetChapters); err != nil {
		return fmt.Errorf("切分章节失败: %w", err)
	}

	chapters, err := env.novelSvc.GetChapters(env.baseCtx, novelID)
	if err != nil {
		return fmt.Errorf("获取章节列表失败: %w", err)
	}

	fmt.Printf("上传并解析完成！\n  ResourceID: %s\n  NovelID: %s\n  Chapters: %d\n  FileSize: %d bytes\n",
		result.ResourceID, novelID, len(chapters), result.FileSize)
	return nil
}

