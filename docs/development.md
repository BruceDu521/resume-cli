# 开发执行与恢复入口

状态：2026-09-20，用户已授权开发、确认默认中文和英文切换，要求广覆盖、优先不联网的 Go 单元测试。用户已提供本任务 Jev/Gemini/DeepSeek key，并授权真实测试；OpenAI/Kimi 未提供。

## 恢复时先读

AGENTS.md、README.md、architecture.md、evaluation.md，以及本机 .local/requirements-source.md、requirements-checklist.md、session-notes.md。原题保密，不进入公开仓库。公开 checkout 缺少 .local 是正常情况。再看 Git 和实际代码，避免将旧计划当现状。

## 已确认

- Go + Cobra + 本地 Poppler，先结构化、再 Jev 判断、代码评分及报告，允许多模型。
- 五项增强全部核心：输出、mock、JSON 修复、日志、Dockerfile 或 Makefile；当前两种构建文件都实现。
- Gemini 与 DeepSeek 比较质量和实际速度，接近时优先低成本 DeepSeek。成本优先的评分可先用 DeepSeek，但两家均有公开技能数组遗漏，暂不设置全局默认。
- OpenAI / Kimi 为独立完整对照，不是只做 fallback，也不当真值。
- 小接口、复用标准机制，不堆框架。不访问其他项目的 key。

## 已完成

- 本地仓库、原题转录和需求清单、设计文档。
- Go 模块，parse/extract/score、输出、mock、中英文、日志、有限 JSON 修复。
- 来源校验、领域评分、模板和 AI 报告、独立基线、缓存、context/重试、用量/成本。
- 四家生成适配器及 Jev 批量 Choice，内存 HTTP 契约测试通过。
- 合成 PDF、12 个开发评测案例和 4 个补充回归案例、默认 dry-run 的评测脚本；mock 评测链路通过。
- 全部 Go 测试及 race/vet 通过：总覆盖率 87.5%，domain 96.1%，ai 90.5%。不含实际模型调用。
- 本地二进制和 Docker 构建；三个命令在禁网 Docker 运行通过。解决了 Debian 缺 poppler-data 导致中文文本为空的问题。
- examples/ 保存真实执行产生的 mock 输出和合成输入的真实模型输出，文件名明确区分。
- JSON 修复器额外完成约 10 秒 fuzz，478,049 次输入执行，无失败；生成种子仅保存在本机 Go fuzz 缓存。

## 当前验证结论与下一步

最终 96 次真实评分和 10 项提取/报告/缓存检查完成，见 [实测报告](evaluation-results-2026-09-20.md)。DeepSeek/Gemini 端到端中位耗时 2.59/3.77 秒；两家都有 skills 遗漏，分类也存在差异。没有独立最终保留集，不把成功率当准确率。

1. OpenAI/Kimi 待本任务 key 后做独立对照；不可自行发现其他凭据。
2. 提取字段完整性是已知质量缺口；扩充新的独立保留集，再判断改动是否有效，不能只调到回归样例通过。
3. 当前无需重复真实调用。代码、离线测试、容器和文档已验收，后续准备演示录像及公开交付。尚未创建 remote、push 或提交招聘材料。

## 本机开发环境

Go 1.25.5，macOS Poppler 26.04.0；Docker 使用本地 OrbStack，Debian 镜像 Poppler 22.12.0。Go 模块缓存 /private/tmp/resume-cli-gomod，构建缓存 /private/tmp/resume-cli-gobuild；依赖已下载，测试 GOPROXY=off。

本机 hook 要求 Go 测试命令以 CLAUDE_APPROVED=1 开头；用户已授权本项目 ./... 的安全离线测试。运行前仍应说明范围；不能把该许可扩展为真实 API、数据库或生产访问。

## 尚存限制

无 OCR、无确定性任期区间计算；复杂 PDF 排版仅基础验证；固定 Choice 等级及评分权重是明确的应用政策，不是外部题目指定。生成模型与 Jev 已实测，语义和字段完整性仍需人工核对。阶段缓存当前 candidate:v5/job:v5；合同变化时必须升级 key。

每轮结束同步此文、README 与本地 session notes，不把未实测能力写成成功结论。
