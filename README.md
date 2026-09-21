# resume-cli

用 Go 编写的 PDF 简历解析与岗位匹配 CLI。PDF 在本地由 Poppler 提取文本；`extract` 和 `score` 各调用一个所选模型，输出经过校验的 JSON。默认中文，支持英文报告。

当前使用单模型评分。普通 `extract` 始终只做全文信息提取。支持 DeepSeek、Gemini、Kimi、OpenAI 和 Anthropic（Claude）；OpenAI/Claude 通过离线协议测试，尚无真实 key 实测。历史对照结果保留，不代表当前版本的性能。当前提取设计与验证见 [全文提取说明](docs/extraction-design.md)。

## 安装与演示

依赖 Go 1.25.5 或更新兼容版本、Poppler（Linux 还需要 poppler-data）。

```sh
# macOS
brew install go poppler
# Debian / Ubuntu
# sudo apt-get install poppler-utils poppler-data
make build
bin/resume-cli --help

# 以下命令不需要 API key
bin/resume-cli parse testdata/resume-zh.pdf
bin/resume-cli extract testdata/resume-zh.pdf --mock
bin/resume-cli score testdata/resume-zh.pdf --jd testdata/jd.txt --mock
```

mock 只识别随项目提供的中英文合成简历及对应 JD；输出 `mock: true`，不会伪装成对任意真实简历的分析。

## 配置

复制 `.env.example` 并填入所选供应商的 key。CLI 不自动读取 dotenv；自行通过可信 shell 或运行环境注入。不要将密钥放进命令行参数。

```sh
cp -n .env.example .env
chmod 600 .env
# 编辑后加载自己创建的配置
set -a
. ./.env
set +a
```

| 变量 | 作用 |
| --- | --- |
| `RESUME_AI_PROVIDER` | `deepseek`、`gemini`、`kimi`、`openai`、`anthropic`（别名 `claude`）；必须明确选择 |
| `RESUME_AI_API_KEY` | 所选供应商的唯一密钥；更换供应商时同步更换 |
| `RESUME_AI_MODEL` | 可选模型 ID 覆盖 |
| `RESUME_AI_BASE_URL` | 可选 HTTPS API 地址覆盖，禁止重定向 |
| `RESUME_CLI_LANG` | 可选：`zh` / `en`，覆盖界面语言，不改变报告语言 |
| `RESUME_LOG_LEVEL` | debug / info / warn / error，默认 info |

模型预设为 DeepSeek `deepseek-flash`、Gemini `gemini-3.8-flash`、Kimi 开放平台 `kimi-k3`、OpenAI `gpt-6-astra`、Claude `claude-sonnet-5`。Kimi Code 订阅使用 `k3` 和独立端点，不能混用开放平台 key：

```sh
# 已注入对应供应商的 RESUME_AI_API_KEY
bin/resume-cli score resume.pdf --jd jd.txt --provider kimi \
  --model k3 --base-url https://api.kimi.com/coding/v1
```

OpenAI 与 Claude 使用相同的环境变量名，分别注入对应厂商 key：

```sh
bin/resume-cli score resume.pdf --jd jd.txt --provider openai --model gpt-6-astra
bin/resume-cli score resume.pdf --jd jd.txt --provider anthropic --model claude-sonnet-5
# --provider claude 与 anthropic 等价
```

Claude 使用原生 Messages API 与 output_config.format；OpenAI 使用 Chat Completions 的严格 JSON Schema。模型需支持相应结构化输出接口，支持厂商不代表兼容其所有历史型号。当前支持和单次费用见 [厂商与成本](docs/providers-and-cost.md)。

所有生成供应商统一使用 `RESUME_AI_API_KEY`，配合 `RESUME_AI_PROVIDER` 和可选的 `RESUME_AI_MODEL` 即可运行，无需模式参数。

## CLI 命令

```sh
bin/resume-cli parse resume.pdf --output resume.txt
bin/resume-cli extract resume.pdf --provider deepseek --output resume.json
bin/resume-cli score resume.pdf --jd jd.txt --provider gemini \
  --output result.json --stats usage.json
bin/resume-cli score resume.pdf --jd jd.txt --provider deepseek --lang en
```

| 参数 | 行为 |
| --- | --- |
| `--output <path>` | 保存结果；parse 为文本，其他为 JSON |
| `--force` | 允许替换输出，禁止覆盖输入或输出别名 |
| `--mock` | 合成样例离线演示 |
| `--lang zh\|en` | 默认 zh；切换评论、面试问题语言，来源不翻译 |
| `--provider` / `--model` / `--base-url` | 覆盖对应环境变量 |
| `--cache-dir <dir>` | extract 使用的可选私有缓存，24 小时有效 |
| `--stats <path>` | 保存成功及失败调用的耗时、token 和估算费用 |
| `--timeout <duration>` | 完整命令默认 90s，最多 10m |

输出文件权限 0600，默认不可覆盖。stdout 只有结果，日志走 stderr；不记录 key、完整简历或模型原始响应。

## 帮助与常见错误

`resume-cli --help` 列出命令用途、参数示例及环境变量；`resume-cli score --help` 提供评分命令示例。`completion` 是 CLI 框架附带的 Shell 补全脚本生成功能，本项目不提供该命令。

先检查简历和 JD，再配置或调用 AI。常见输入错误根据运行环境显示中英文说明和处理建议，写入 stderr，退出码为 1，stdout 不混入错误信息：

```text
resume-cli: 岗位描述（JD）："jd.none"：文件不存在，请检查路径和文件名。
resume-cli: 岗位描述（JD）："jd.empty"：文件为空或仅含空白，请填写岗位描述后重试。
resume-cli: 简历 PDF："resume.pdf"：PDF 无法解析，可能已损坏或格式不受支持；请确认能正常打开，并重新导出 PDF。
```

还会检查目录误用、读取权限、非 PDF、空 PDF、加密文件、扫描件无文本、UTF-8 编码、文件/文本大小及解析工具缺失。JD 必须是纯文本，PDF 需先转为文本。输出目录不存在、无写入权限或已有文件也会给出提示。常见文件错误不会直接显示 `stat`、内部临时路径或子进程退出信息。

帮助与常见输入错误在运行时选择语言，不需要分别编译。优先级为 `RESUME_CLI_LANG` → `LC_ALL` → `LC_MESSAGES` → `LANG`（取首个非空值）；`zh_CN.UTF-8`、`zh_TW` 等中文 locale 显示中文，英文、C/POSIX、其他或未设置的 locale 显示英文。不会修改文件名、简历内容或 AI 输出；底层技术诊断和日志保持原有英文。

```sh
RESUME_CLI_LANG=en bin/resume-cli --help
RESUME_CLI_LANG=zh bin/resume-cli extract missing.pdf
# 英文界面，报告仍默认中文
RESUME_CLI_LANG=en bin/resume-cli score resume.pdf --jd jd.txt
# 中文界面，生成英文报告
RESUME_CLI_LANG=zh bin/resume-cli score resume.pdf --jd jd.txt --lang en
```

界面语言与报告语言完全独立；报告始终默认中文，仅由 `--lang en` 切换。

## 示例输入与输出

输入见 `testdata/resume-zh.pdf`、`testdata/jd.txt`；示例人物和联系方式均为合成。`extract` 输出姓名、电话、邮箱、城市、education 和 skills；缺失内容为 `""` / `[]`，不填造事实。

[中文 extract 示例](examples/extract-zh.mock.json)、[中文 score 示例](examples/score-zh.mock.json)、[英文 score 示例](examples/score-en.mock.json)。mock 评分主字段如下：

```json
{
  "overall_score": 83,
  "skill_score": 100,
  "experience_score": 50,
  "education_score": 100
}
```

默认完整结果还包含 comment、interview_questions、policy_version、language 和 mock。上述分数是固定演示，不能视为实际模型评测结果。

## 技术选择与流程

Go + Cobra 负责 CLI、文件边界、HTTP、取消和 JSON 校验；Poppler 负责本地 PDF 文本提取。无 Agent 框架、数据库或服务端依赖。

- `parse`：只在本机读取 PDF，不调用 AI。
- `extract`：把完整 PDF 提取文本传给模型，直接生成姓名、联系方式、城市、education、skills。只要求公开 JSON 结构，不生成 facts、行号引用或评分。技能根据实际工作/项目/技能描述整理，允许归纳名称，缺项不编造。
- `score`：把完整简历文本和完整 JD 传给模型，直接返回四项 0–100 整数分数、评语和面试问题，不要求行号、逐字引文、事实目录或中间匹配数组。本地校验结构、分数范围和非空报告；不截断职责/要求。无效结果最多纠正一次；纠正调用失败仍保留初次校验原因。

默认评分使用 `model-assessment-v1`：四项分数均是模型结合岗位重点作出的评价，没有固定加权公式，也没有代码保证语义或覆盖完整性；必须结合评语人工复核。不能与旧版逐项证据评分或不同模式的数值直接比较。


单模型、一次请求并不保证每次结果相同；模型判断和自由文本仍需质量验证。详细设计见 [架构文档](docs/architecture.md)。

## 测试、评测与 Docker

离线 Go 单测/race/vet 持续验证，最新结果见开发记录。Go 单元测试使用内存 HTTP 替身，AI/CLI 测试默认禁止真实网络；测试不读取 `.env`。离线验证覆盖完整文本输入、JSON 修复、评分、重试、取消、缓存及输出防覆盖。

```sh
make check
python3 -m unittest discover -s scripts -p 'test_*.py'
# 默认只列计划，不读取 key 或联网
python3 scripts/evaluate.py
# 离线演示评测
python3 scripts/evaluate.py --routes mock --repeats 1 --execute --out .local/eval-new
# 明确准备各供应商私有配置后才执行真实 API
python3 scripts/evaluate.py --routes gemini deepseek --env-dir .local/provider-env \
  --limit 2 --repeats 1 --execute --out .local/eval-live-new

docker build -t resume-cli .
docker run --rm --network none resume-cli score /examples/resume-zh.pdf \
  --jd /examples/jd.txt --mock
```

真实模型测试与单测分开，费用只是按已观察 token 的估算；未知费用不写成零。Kimi Code 为订阅配额，不能冒充按 token 美元账单。多供应商配置方法及检查规则见 [评测协议](docs/evaluation.md)。

## 已实现功能与限制

已实现三个命令、中英文、文件输出、mock、有限 JSON 修复、日志、Makefile 与 Dockerfile。JSON 修复仅处理完整代码围栏、BOM、字符串外尾逗号；不修造业务事实。拒绝重复键、null、未知字段以及过量输入。

- PDF 上限 100 MiB、文本 160 KiB、JD 64 KiB（UTF-8 字节数，不是字符数或要求条数）；扫描件需要 OCR，当前不支持；不解锁加密 PDF。
- 不自动纠正多栏阅读顺序或跨页页眉；完整文本传给模型，不按技能或 JD 条数截断。资源上限用于控制内存和请求开销，超限明确报错。
- extract 校验 JSON 结构和空值约定，不用原文子串检查代替语义判断；因此不能保证模型提取没有遗漏或归纳错误。默认评分只做结构与范围校验，不能证明职责覆盖和评论语义准确。
- 没有确定性任期合并、精确技能年限推导或批量招聘服务。不得把总工龄当技能年限。
- 合成回归集不是独立人工标注准确率；真实复杂文档仅有限测试，不声称生产稳定性。OpenAI/Claude 尚未真实调用，Windows 或高并发未验证。
- 尚未发布公开仓库或演示视频。原题、真实简历、密钥及中间结果均排除 Git。

历史报告：[早期组合评测](docs/evaluation-results-2026-09-20.md)、[单模型与组合比较](docs/evaluation-single-vs-hybrid-2026-09-20.md)、[跨行修复前的真实简历测试](docs/evaluation-real-resume-2026-09-20.md)。这些记录保留失败，不能与新版本测量混算。第三方信息见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
