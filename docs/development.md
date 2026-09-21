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

## 可配置输入上限（最新，2026-09-21）

用户确认默认 PDF100MiB、提取文本128KiB、JD64KiB，增加 --max-pdf-mib / --max-text-kib / --max-jd-kib 正整数参数。限制分别用于 fileio.Read、Poppler stdout 有界写入、JD 文本读取；提前检查无效值和乘法溢出。超限拒绝，不截断。旧200KiB序列化输入限制已移除，防止参数调大后再次被隐藏限制拒绝；JSON转义不等于原始输入增多。模型上下文和输出token上限仍是模型约束，本地字节限制不宣称精确token计数。

## 面试交付 README 与硬上限（最新，2026-09-21）

README 已按需求理解、快速开始、配置、命令、示例、设计取舍、代码结构、测试、限制组织，非对话流水记录。最新资源默认32MiB/64KiB/32KiB，硬上限200MiB/256KiB/128KiB；此前200/256/128只是示例，现已按本轮建议设置为硬上限。零/负数/小数/非法文字/溢出/超过最大值均拒绝，最大值本身允许；源文本不截断。模型token容量独立，不承诺放大资源上限后请求一定被模型接受。

## 2026-09-21 二进制样例分发与交付说明

新增 samples <新目录>，通过根目录 samples.go 的 go:embed 嵌入四份既有合成 PDF/JD，单独分发二进制也可离线导出；禁止覆盖已有目录（含 --force）。只嵌入明确的合成路径，不包含真实简历。真实 PDF 命令仍依赖 Poppler；samples 本身不依赖。新增导出与中英文完整 mock 流程回归。

Makefile 的 test/race/vet 不再强制 GOPROXY=off，首次可正常下载 Go 依赖；测试 HTTP 替身仍禁真实 AI 网络。README 区分依赖下载与模型调用，提供显式离线检查方式、二进制/源码/Docker 使用步骤、运行时 key/只读输入/宿主机输出方法。帮助与 README 明确未指定 RESUME_CLI_LANG 时使用终端 locale。Jev 历史小节区分考虑过的 Score 和实际 Choice 原型，记录配置复杂度、无足够稳定成本速度收益和最终评分问题；保留早期合成测试的局部收益，不混称当前版本结果。

全包 race 测试、vet、make build 通过。独立临时目录仅放二进制，导出/parse/extract/score、英文帮助与默认中文报告均通过；README JSON/本地链接和 diff 检查通过。Docker 构建已实际尝试，但 auth.docker.io/token 返回 EOF，未进入构建，尚未验证最新镜像。没有真实 API 调用，也未改用户 .env/私有输入/其他 session。
