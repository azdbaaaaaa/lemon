package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"lemon/internal/service"
)

var novelUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "上传小说文本文件并创建资源记录",
	Long: `上传小说文本文件到存储系统并创建资源记录。

示例：
  lemon novel upload --file assets/novel/001/xxx.txt --user user123
  lemon novel upload -f ./novel.txt -u user123 --ext txt`,
	RunE: runNovelUpload,
}

func init() {
	uploadFlags := novelUploadCmd.Flags()
	uploadFlags.StringP("file", "f", "", "小说文本文件路径（例如：assets/novel/001/xxx.txt）")
	uploadFlags.StringP("user", "u", "", "资源所属用户ID")
	uploadFlags.String("content-type", "text/plain", "文件内容类型（MIME type）")
	uploadFlags.String("ext", "", "文件扩展名（不含点号，默认根据文件名推断）")
	_ = novelUploadCmd.MarkFlagRequired("file")
	_ = novelUploadCmd.MarkFlagRequired("user")
}

// runNovelUpload 执行上传小说文件命令
func runNovelUpload(cmd *cobra.Command, args []string) error {
	env, err := newNovelCLIEnv()
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

	fmt.Printf("上传成功！\n  ResourceID: %s\n  FileSize: %d bytes\n", result.ResourceID, result.FileSize)
	return nil
}

