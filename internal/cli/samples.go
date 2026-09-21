package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	samples "resume-cli"
	"resume-cli/internal/fileio"
	"resume-cli/internal/i18n"

	"github.com/spf13/cobra"
)

func samplesCommand(lang string) *cobra.Command {
	tr := func(s string) string { return i18n.Text(lang, s) }
	return &cobra.Command{
		Use:     "samples" + tr(" <新目录>"),
		Short:   tr("导出内置中英文合成简历和 JD，无需源码、密钥或网络"),
		Example: "  resume-cli samples demo-inputs\n  resume-cli score demo-inputs/resume-zh.pdf --jd demo-inputs/jd.txt --mock",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return i18n.New("请提供一个新的样例目录，例如 resume-cli samples demo-inputs")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if err := os.Mkdir(dir, 0700); err != nil {
				if errors.Is(err, os.ErrExist) {
					return &fileio.Error{Path: dir, Message: "样例目录已存在，请指定新目录；不会覆盖已有内容。", Cause: err}
				}
				return fileio.WriteError(dir, err)
			}
			entries, err := samples.Files.ReadDir("testdata")
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := cmd.Context().Err(); err != nil {
					return err
				}
				data, err := samples.Files.ReadFile("testdata/" + entry.Name())
				if err != nil {
					return err
				}
				if err := fileio.Write(filepath.Join(dir, entry.Name()), data, false); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), tr("已导出中英文样例到 %s\n"), dir)
			return err
		},
	}
}
