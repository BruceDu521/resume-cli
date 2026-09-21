package cli

import (
	"github.com/spf13/cobra"
	"resume-cli/internal/i18n"
)

const helpTemplate = `{{with (or .Long .Short)}}{{.}}
{{end}}
用法：
  {{if .Runnable}}{{.UseLine}}{{else}}{{.CommandPath}} <命令> [参数]{{end}}
{{if .HasAvailableSubCommands}}
命令：{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}
{{end}}{{if .HasExample}}
示例：
{{.Example}}
{{end}}{{if .HasAvailableLocalFlags}}
参数（均可省略，除非标为必填）：
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}
通用参数：
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}
AI 配置（extract / score 使用；parse 和 --mock 不需要密钥）：
  RESUME_AI_PROVIDER   厂商：deepseek / gemini / kimi / openai / anthropic（或 claude）
  RESUME_AI_API_KEY    对应厂商的 API key，仅通过环境变量传入
  RESUME_AI_MODEL      可选；不设置时使用该厂商的默认型号
  RESUME_AI_BASE_URL   可选；通常无需修改
  RESUME_LOG_LEVEL     可选；debug / info / warn / error，默认 info

已配置环境变量时，无需再传 --provider 或 --model；命令行参数优先。
程序不会自动读取 .env。使用自己配置的 .env：set -a; source .env; set +a
界面语言：RESUME_CLI_LANG > LC_ALL > LC_MESSAGES > LANG。
zh 开头使用中文，其他语言或未设置时使用英文；可设 RESUME_CLI_LANG=zh 或 en。
报告语言独立：默认中文，仅由 --lang en 切换为英文。

输入边界：PDF 100 MiB；提取文本 160 KiB；JD 64 KiB（UTF-8 字节数，不是字符数）。
超限会报错，不会截断；不限制技能或岗位要求的条数。
{{if .HasAvailableSubCommands}}
运行 resume-cli <命令> --help 查看该命令的用法和示例。
{{end}}`

const helpTemplateEN = `{{with (or .Long .Short)}}{{.}}
{{end}}
Usage:
  {{if .Runnable}}{{.UseLine}}{{else}}{{.CommandPath}} <command> [options]{{end}}
{{if .HasAvailableSubCommands}}
Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}
{{end}}{{if .HasExample}}
Examples:
{{.Example}}
{{end}}{{if .HasAvailableLocalFlags}}
Options (optional unless marked required):
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}
Global options:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}
AI configuration (extract / score only; parse and --mock need no key):
  RESUME_AI_PROVIDER   deepseek / gemini / kimi / openai / anthropic (alias: claude)
  RESUME_AI_API_KEY    API key for the selected provider; environment variable only
  RESUME_AI_MODEL      Optional; defaults to the selected provider's default model
  RESUME_AI_BASE_URL   Optional; usually unnecessary
  RESUME_LOG_LEVEL     Optional; debug / info / warn / error; defaults to info

When environment variables are set, --provider and --model can be omitted.
Command-line options take precedence. The CLI does not automatically load .env.
To load your own .env in a POSIX shell: set -a; source .env; set +a

Interface language: RESUME_CLI_LANG > LC_ALL > LC_MESSAGES > LANG.
Locales starting with zh use Chinese; other/unset locales use English.
Override with RESUME_CLI_LANG=zh or en.
Report language is independent: Chinese by default; use --lang en for English.

Input limits: PDF 100 MiB; extracted text 160 KiB; JD 64 KiB (UTF-8 bytes, not characters).
Oversized input is rejected, never truncated. No limits on skill/requirement counts.
{{if .HasAvailableSubCommands}}
Run resume-cli <command> --help for command-specific usage and examples.
{{end}}`

func configureHelp(root *cobra.Command, lang string) {
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpTemplate(helpTemplate)
	if lang == "en" {
		root.SetHelpTemplate(helpTemplateEN)
	}
	root.PersistentFlags().BoolP("help", "h", false, i18n.Text(lang, "查看命令用法、参数和配置说明"))
	root.SetHelpCommand(&cobra.Command{
		Use: i18n.Text(lang, "help [命令]"), Short: i18n.Text(lang, "查看帮助，例如 resume-cli help score"),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _, err := root.Find(args)
			if err != nil {
				return err
			}
			return target.Help()
		},
	})
}
