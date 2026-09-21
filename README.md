# resume-cli

用 Go 编写的 PDF 简历解析与岗位匹配 CLI。PDF 在本地由 Poppler 提取文本；`extract` 和 `score` 各调用一个所选模型，输出经过校验的 JSON。默认中文，支持英文报告。

当前默认单模型；保留 `--pipeline jev` 显式选择模型 + Jev 评分。普通 `extract` 始终只做全文信息提取。支持 DeepSeek、Gemini、Kimi 和 OpenAI 适配器；OpenAI 尚未实测。历史对照结果保留，不代表当前版本的性能。当前提取设计与验证见 [全文提取说明](docs/extraction-design.md)。

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
| `RESUME_AI_PROVIDER` | `deepseek`、`gemini`、`kimi` 或 `openai`；必须明确选择 |
| `RESUME_AI_API_KEY` | 所选供应商的唯一密钥；更换供应商时同步更换 |
| `RESUME_AI_MODEL` | 可选模型 ID 覆盖 |
| `RESUME_AI_BASE_URL` | 可选 HTTPS API 地址覆盖，禁止重定向 |
| `RESUME_AI_PIPELINE` | 默认 `single`；显式设为 `jev` 才使用组合评分 |
| `TYPESAFE_API_KEY` / `RESUME_JEV_MODEL` / `TYPESAFE_BASE_URL` | 可选 Jev 配置，仅 Jev 评分读取；模型默认 `jev-1.13.0` |
| `RESUME_LOG_LEVEL` | debug / info / warn / error，默认 info |

模型预设为 DeepSeek `deepseek-flash`、Gemini `gemini-3.8-flash`、Kimi 开放平台 `kimi-k3`、OpenAI `gpt-6-astra`。Kimi Code 订阅使用 `k3` 和独立端点，不能混用开放平台 key：

```sh
# 已注入对应供应商的 RESUME_AI_API_KEY
bin/resume-cli score resume.pdf --jd jd.txt --provider kimi \
  --model k3 --base-url https://api.kimi.com/coding/v1
```

所有生成供应商统一使用 `RESUME_AI_API_KEY`。默认 single 不读取 Jev key；`--pipeline jev` 额外需要 `TYPESAFE_API_KEY`，可通过 `--jev-model` 覆盖模型。旧 `--pipeline hybrid` 兼容为 jev；`--report` 不再提供，Jev 路线使用本地模板报告。

## CLI 命令

```sh
bin/resume-cli parse resume.pdf --output resume.txt
bin/resume-cli extract resume.pdf --provider deepseek --output resume.json
bin/resume-cli score resume.pdf --jd jd.txt --provider gemini \
  --output result.json --stats usage.json
bin/resume-cli score resume.pdf --jd jd.txt --provider deepseek --lang en
# 可选组合模式；另需 Jev key
bin/resume-cli score resume.pdf --jd jd.txt --provider deepseek --pipeline jev
```

| 参数 | 行为 |
| --- | --- |
| `--output <path>` | 保存结果；parse 为文本，其他为 JSON |
| `--force` | 允许替换输出，禁止覆盖输入或输出别名 |
| `--mock` | 合成样例离线演示 |
| `--lang zh\|en` | 默认 zh；切换评论、面试问题语言，来源不翻译 |
| `--provider` / `--model` / `--base-url` | 覆盖对应环境变量 |
| `--pipeline single\|jev` / `--jev-model` | 默认 single；显式选择 Jev 路线及模型 |
| `--cache-dir <dir>` | extract 或 Jev 路线使用的可选私有缓存，24 小时有效 |
| `--stats <path>` | 保存成功及失败调用的耗时、token 和估算费用 |
| `--timeout <duration>` | 完整命令默认 90s，最多 10m |

输出文件权限 0600，默认不可覆盖。stdout 只有结果，日志走 stderr；不记录 key、完整简历或模型原始响应。

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

完整结果还包含 comment、interview_questions、findings（逐项要求、状态与原文证据）、not_required、policy_version、language 和 mock。上述分数是固定演示，不能视为实际模型评测结果。

## 技术选择与流程

Go + Cobra 负责 CLI、文件边界、HTTP、取消和 JSON 校验；Poppler 负责本地 PDF 文本提取。无 Agent 框架、数据库或服务端依赖。

- `parse`：只在本机读取 PDF，不调用 AI。
- `extract`：把完整 PDF 提取文本传给模型，直接生成姓名、联系方式、城市、education、skills。只要求公开 JSON 结构，不生成 facts、行号引用或评分。技能根据实际工作/项目/技能描述整理，允许归纳名称，缺项不编造。
- `score` 默认 single：一次模型请求完成简历/JD 整理、逐项判断、评论及面试问题；本地核验来源和引用，再统一算分。结构或来源校验失败时最多从原输入纠正一次。
- `score --pipeline jev`：生成模型分别整理简历证据和 JD，再由 Jev 判断，代码算分并用本地模板报告。
- 仅评分内部使用来源块，保留 PDF 提取文本行号。事实用 `block_id` 和 `end_block_id` 指定最多 16 个连续块；单行可省略或留空 end_block_id。引文必须是该范围的连续原文，允许空白差异，不能跳行或改写。输出保留完整范围上下文。这些 b1/b2 编号由代码添加，不是 PDF 原生段落；extract 不使用它们。已取消原先人为设置的 64 条证据上限，不限制公开技能/学历条数。

权重是本项目明确制定的匹配政策，非题目指定：满足=100、部分满足=50、明确不符=0、未体现=0；必需项权重 2，优先项 1；技能/经历/教育权重 50%/35%/15%。未要求的维度从总分分母移除，并标记 not_required；该维度数字 100 是兼容占位，不代表实际能力满分。

单模型、一次请求并不保证每次结果相同；模型判断、引用选择和自由文本仍需质量验证。详细设计见 [架构文档](docs/architecture.md)。

## 测试、评测与 Docker

离线 Go 单测/race/vet 持续验证，最新结果见开发记录。Go 单元测试使用内存 HTTP 替身，AI/CLI 测试默认禁止真实网络；测试不读取 `.env`。离线验证覆盖来源范围、否定语境、JSON 修复、评分、重试、取消、缓存及输出防覆盖。

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

已实现三个命令、中英文、文件输出、mock、有限 JSON 修复、日志、Makefile 与 Dockerfile。JSON 修复仅处理完整代码围栏、BOM、字符串外尾逗号；不修造业务事实。拒绝重复键、null、未知字段、无效引用以及过量输入。

- PDF 上限 20 MiB、文本 160 KiB、JD 64 KiB；扫描件需要 OCR，当前不支持；不解锁加密 PDF。
- 跨行来源支持连续范围，不自动理解所有多栏顺序、跨页页眉或扫描版面。评分每条引用最多 16 行，最多 24 项 JD 要求；这些是当前工程边界，不是题目要求。Jev 超出本地请求大小预算会明确报错，不静默截断简历。
- extract 校验 JSON 结构和空值约定，不用原文子串检查代替语义判断；因此不能保证模型提取没有遗漏或归纳错误。评分另做原文引用校验，但不能证明引用充分或评论语义准确。
- 没有确定性任期合并、精确技能年限推导或批量招聘服务。不得把总工龄当技能年限。
- 合成回归集不是独立人工标注准确率；真实复杂文档仅有限测试，不声称生产稳定性。未测试 OpenAI、Windows 或高并发。
- 尚未发布公开仓库或演示视频。原题、真实简历、密钥及中间结果均排除 Git。

历史报告：[早期组合评测](docs/evaluation-results-2026-09-20.md)、[单模型与组合比较](docs/evaluation-single-vs-hybrid-2026-09-20.md)、[跨行修复前的真实简历测试](docs/evaluation-real-resume-2026-09-20.md)。这些记录保留失败，不能与新版本测量混算。第三方信息见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
