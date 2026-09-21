# 开发恢复说明

## 最新决定（2026-09-21）

用户明确要求从本仓库删除 Jev，另一个 session 在独立目录维护该实验。本目录只保留单模型；移除模式参数、专用 key 读取、Candidate/Job/Matcher、证据聚合及旧报告模板。不要修改另一个目录，也不要改动用户的 .env、.env.bak 或私有凭据文件。

extract 使用完整文本 + ResumeSchema。score 使用完整文本 + JD，直接生成四项分数、comment、interview_questions，policy=model-assessment-v1。保持现有提示词和一次纠正重试；纠正请求失败也保留初次校验原因。结构校验不能代表语义或要求覆盖正确。

## 测试与配置

本机 Go 依赖缓存 /private/tmp/resume-cli-gomod 和 /private/tmp/resume-cli-gobuild，可用 GOPROXY=off。本机测试 hook 要求 CLAUDE_APPROVED=1。Go 单测使用内存替身并阻止真实网络，不读取 .env 或数据库。

统一 RESUME_AI_API_KEY，provider/model/base URL 可通过环境配置或参数覆盖。任务专用供应商配置保存在 .local/provider-env；不得发现其他项目凭据。Kimi Code 使用 k3 和 https://api.kimi.com/coding/v1。OpenAI/Claude 尚无真实 key 验证。

最近实际单模型评分 DS 63、Gemini 76、Kimi 75，各一次成功。详见 .local/session-notes.md；不同版本不得混算速度、成本或准确率。移除 Jev 不需要重新执行付费评测。

## 交付边界

- 保留用户 root resume.pdf、jd.txt 等本地材料，不纳入 Git。
- 历史评测仅作决策记录，旧字段和命令不适用于当前版。
- 技能归纳、评论语义及复杂 PDF 版面仍需人工复核。
- 公开仓库、视频与招聘提交尚未完成，无 remote/push。
- Dockerfile 上次重建曾因 Docker Hub auth EOF 未完成；本次本机测试不代表镜像已重建。
- OCR、Windows、高并发、确定性任期合并与技能年限未实现或验证。

## CLI 输入错误与帮助（2026-09-21）

常见文件错误使用 fileio.Error：对外显示中文原因与操作建议，Unwrap 保留底层 cause 供 errors.Is/As 使用。PDF 错误带简历角色，JD 错误带岗位描述角色；不打印 Poppler stderr/临时路径。CLI 在读取 key 前验证本地输入，并复用解析结果避免重复运行 Poppler。帮助页补齐命令介绍、例子、所有配置变量与优先级，禁用默认 completion。

bounded 使用命名 buffer 而非嵌入 bytes.Buffer，避免 io.Copy 通过继承的 ReadFrom 绕过 Write 大小限制。离线测试覆盖真实子进程超限、损坏/加密/无文字、错误链保留及常见 CLI 输入失败。

## 运行时界面语言（2026-09-21）

界面语言优先级 RESUME_CLI_LANG > LC_ALL > LC_MESSAGES > LANG，首个非空值为中文 locale 时用 zh，否则含 C/POSIX、未设置与不支持的语言统一回退 en。帮助和常见用户输入错误双语；底层技术诊断与日志保留英文。报告仍独立默认 zh，--lang en 切换英文。无编译开关或全局可变语言状态。

i18n 使用完整消息目录和带参数的错误对象，终端出口渲染；不在格式化后的报错中搜索替换中文，避免改变文件名。errors.Is/As 原因链保留。离线回归覆盖 locale 优先级、中英文帮助/错误、界面与报告语言四种组合、JD 64 KiB UTF-8 字节边界。

## 带作品集的 PDF 大小（2026-09-21）

用户指出设计类求职者可能在简历中放入作品。PDF 文件上限提高到 100 MiB；提取文本 160 KiB、JD 64 KiB、序列化模型输入 200 KiB 维持独立限制。这里只扩大可读取文件大小，仍为文本解析，不包含图片视觉分析或 OCR。历史记录中的 20 MiB 为旧值。
