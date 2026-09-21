package cli

import "github.com/spf13/cobra"

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
score 的 --lang 只切换报告语言；帮助和常见输入错误使用中文。
{{if .HasAvailableSubCommands}}
运行 resume-cli <命令> --help 查看该命令的用法和示例。
{{end}}`

func configureHelp(root *cobra.Command) {
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpTemplate(helpTemplate)
	root.PersistentFlags().BoolP("help", "h", false, "查看命令用法、参数和配置说明")
	root.SetHelpCommand(&cobra.Command{
		Use: "help [命令]", Short: "查看帮助，例如 resume-cli help score",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _, err := root.Find(args)
			if err != nil {
				return err
			}
			return target.Help()
		},
	})
}
