# 架构与实现

2026-09-21：本仓库仅保留单模型流程。Jev 的独立实验由另一个目录维护；历史评测不能代表当前版本。

## 主流程

- `parse`：有界读取本地 PDF → 私有临时文件 → 可取消的 Poppler 子进程 → 完整文本。没有模型调用。
- `extract`：完整文本 + ResumeSchema → 所选模型 → JSON 与字段校验 → Resume。可显式启用 24 小时私有缓存。
- `score`：完整简历文本 + JD + 六字段 Schema → 所选模型 → 四项 0–100 整数分数、评语、面试问题 → 结构/范围/非空校验。独立于 extract，不复用其缓存。

两项 AI 任务共用 Generator 和 decodeChecked：正常一次生成；无效 JSON 或字段校验失败最多再纠正一次，使用原输入和固定校验原因，不回传未信任响应。纠正请求失败时保留初次校验原因。所有调用均记录用量。

## 模块

| 模块 | 职责 |
| --- | --- |
| cmd/resume-cli | 信号取消和退出码 |
| internal/cli | Cobra、环境配置、依赖组装、日志、stats |
| internal/app | Parse/Extract/Score 用例，Parser/Extractor/Evaluator 小接口 |
| internal/domain | Document、Resume、提取字段校验 |
| internal/pdf | 本地 Poppler、文件与文本限制、取消 |
| internal/ai | 提示词/schema、五个供应商适配器、mock、HTTP、用量 |
| internal/i18n | 运行时界面语言检测、消息目录、错误渲染；独立于报告语言 |
| internal/report | Evaluation 与最终报告、分数和报告校验 |
| internal/cache | extract 显式私有缓存 |
| internal/fileio / jsonutil | 有界输入、原子输出、有限 JSON 修复 |

Document 保留完整 text/hash 和解析行信息；模型只接收完整文本，不接收编号块。Resume 包含姓名、联系方式、城市、education、skills；缺项为空字符串或数组。Evaluation 包含四项分数、comment、interview_questions，最终报告增加 policy_version、language、mock。

## 提示词与模型

extract 按完整工作、项目和技能描述归纳，不猜测缺失信息。score 同时考虑职责与要求、合并重复条件、区分必需与优先，区分未体现和不具备，不凭总工龄推断技能年限。提示词不保证语义正确，代码不校验 JD 覆盖完整性。

评分策略 `model-assessment-v1`：模型决定四项分数，无固定加权公式。字段名固定英文，事实保留输入语言，评语和问题默认中文、可选英文。mock 返回中英文固定合成示例，不执行真实推断。

provider/model/base URL 可由环境或参数选择，统一 RESUME_AI_API_KEY。Gemini 使用 Interactions，Claude 使用原生 Messages，其余使用各自 Chat Completions。Kimi Code 与开放平台端点和型号不同。

Claude HTTP Schema 移除不支持的数值上下界，本地仍校验范围；只读文本块，拒绝截断/拒答/工具调用终态。OpenAI 使用严格 JSON Schema，只向推理型号发送 reasoning_effort。OpenAI/Claude 目前仅通过离线协议测试。

## 缓存、安全和限制

extract 缓存键 resume:v1，包含 provider/model/endpoint 与输入哈希，有效 24 小时。score 无缓存。密钥不进入参数、日志、缓存或报告。

PDF 100 MiB、文本 160 KiB、JD 64 KiB、序列化模型输入 200 KiB、AI 响应 2 MiB。超过资源限制明确报错，不按技能或要求条数截断。HTTP 单次 60s，命令默认 90s；429/529/502/503/504 最多三次，取消传播，不重试不明网络故障或 401。HTTPS 且禁止重定向。

输出默认不可覆盖，不能覆盖输入或文件别名，私有文件 0600。JSON 拒绝重复键、null、缺失和未知字段；仅修复完整围栏、BOM、字符串外尾逗号。

单元测试使用内存 HTTP 替身，默认 transport 禁网，不读取凭据。真实 API 验证独立执行。扫描 PDF、复杂多栏顺序和任意长文的语义完整性并未解决。校验通过不代表评分准确。
