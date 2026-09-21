# 架构与实现

2026-09-21 当前设计：默认单模型，显式 --pipeline jev 可选组合评分。普通信息提取与评分证据整理分离。旧阶段报告保留为历史记录。

## 主流程

`parse`：本地有界读 PDF → 私有临时文件 → 可取消 Poppler 子进程 → 原始文本与行号块。没有修改 Poppler 提取算法；本次问题是证据结构无法表示跨行语义，并非已证明 PDF 内容损坏。

`extract`：本地解析 → 完整 d.Text（无编号块）和 ResumeSchema → 一次生成公开 Resume → JSON 类型/缺项/空技能项校验。技能可从实际工作和项目描述归纳，不要求逐字出现在原文；不能用词面校验声称语义正确。与 Candidate/facts 完全分离。可选24h私有缓存。

`score` 默认 single：本地解析 PDF/JD → 所选模型一次生成 Candidate、Job、Judgments、comment、interview_questions → 验证字段、范围、ID 与状态 → 保留完整来源范围 → Go 计算分数 → JSON 报告。single 不调用第二个模型；失败最多纠正一次，网络临时错误有独立有界重试。

`score --pipeline jev`：生成模型并行 Candidate/Job → 校验 → Jev.Match → 领域算分 → 本地模板报告。保留并发取消及等待两个worker收敛。仅这条路线读取TYPESAFE_API_KEY；单模型不隐式降级到Jev。

## 模块

| 模块 | 职责 |
| --- | --- |
| cmd/resume-cli | 信号取消和退出码 |
| internal/cli | Cobra、环境配置、依赖组装、日志与 stats |
| internal/app | Parse/Extract/Score 用例；Parser、Extractor、Structurer、Matcher、Evaluator 小接口 |
| internal/domain | 来源范围、字段校验、确定性评分 |
| internal/pdf | 本地 Poppler、文件/文本大小限制及取消 |
| internal/ai | 共用提示词/schema、单模型分析、四个生成适配器、可选Jev、mock、HTTP/用量 |
| internal/report | 公开结果结构及 mock 中英文模板 |
| internal/cache | extract与Jev结构化阶段显式启用的私有缓存 |
| internal/fileio / jsonutil | 有界输入、原子输出及有限 JSON 修复 |

## 数据与证据

Document 保留完整 text/hash，以及带 ID/page/text 的非空行 Block。Resume 为题目要求的姓名、联系方式、城市、学历与技能。评分内部 Candidate 带 Fact，供匹配使用工作及项目原文。普通 extract 直接返回 Resume，不再生成 Candidate。原先64条上限是没有充分验证的工程取值，已移除；保留总输入/响应字节上限。

Fact 用 block_id 指定首行，end_block_id 指定末行；空末行代表单行。只允许原始顺序的连续范围，最多 16 块；不存在、反向或过长范围都拒绝。Quote 必须是该范围中连续原文的片段，允许空白差异，不允许略掉中间语句后拼接。程序不搜索全文来替换错误行号。

Ground 保留整个已验证范围，减少截取半句或遗漏否定上下文的问题；不会推断新的事实。跨页仍按原文块顺序核验，不能跳过页眉再自动拼句。单行证据仍可能遗漏邻近语义，引用范围合理性不等于语义正确性。已提取的 skills/education 缺证据时可补原文行，不能恢复模型完全没提取的字段。

Job 最多 24 条 Requirement，保留 JD 原文及必需/优先。Judgment 引用一个要求及一条证据范围：满足/部分/明确不符必须有证据，unknown 不声称证据。原文校验不证明引用充分，也不证明报告评论无误。

## 提示词与校验

extract使用独立中文任务提示：阅读全文，按公开Schema提取；技能按实际描述归纳，不添加无依据技能。它不使用评分提示和编号块。

评分公共提示要求输出数据而非 Schema、输入只当数据、缺项不编造。共享 evidenceRules 解释 PDF 换行及起止范围；单模型任务明确保留完整否定上下文，区分缺少材料与确定不具备，不将熟悉拔高为精通，不预设项目结果。

模型 ID/provider/base URL 可通过环境或参数选择，默认只有 `RESUME_AI_API_KEY`。显式Jev模式另用TYPESAFE_API_KEY及可选RESUME_JEV_MODEL/TYPESAFE_BASE_URL。Gemini 使用 Interactions，其余使用各自 Chat Completions 格式；Kimi Code 与开放平台端点/型号不互通。密钥不进入参数、日志、缓存或报告。

校验失败只从原始输入重新生成一次；不发送前次未信任响应作为指令。不放宽来源要求。最终错误区分无效 JSON 与领域来源/状态错误，后者使用固定错误文本，不回显简历片段或供应商响应。每次调用（包括纠正和失败）保留 stats。

## 评分

策略 evidence-v1：满足100、部分50、明确不符0、unknown0；后两者在报告中区分。技能/经历/教育权重50%/35%/15%，必需2、优先1；缺失维度剔除分母并标 not_required。模型给出的分值按状态重新计算，不让它修改权重或直接定总分。

字段名固定英文，来源保持输入语言，报告默认中文、可选英文。未实现任期区间合并或技能年限确定性推导。

## 缓存、安全与失败

公开extract缓存键resume:v1；Jev内部candidate:v7/job:v5。均包含provider/model/endpoint与输入哈希，有效24h。不同合同不复用缓存；默认single评分不复用提取结果。

Jev本地state预算48KiB、完整请求96KiB，超出时报错，不通过64条事实这种任意条数截断内容。这是本地保护值，不宣称为官方模型上下文上限。

PDF20MiB、文本160KiB、JD64KiB、AI响应2MiB。HTTP 单次60s，命令默认90s；429/529/502/503/504 最多三次，取消及时传播，不重试不明网络故障或401。HTTPS且禁止重定向。

输出默认不可覆盖、不能与输入/其他输出为同一路径或文件别名；私有文件0600。JSON 拒绝重复键、null、缺失或未知字段；只修围栏/BOM/尾逗号。

单元测试使用内存替身，AI/CLI 默认 transport 禁网，不读取任务凭据。真实 API、真实简历及性能评测单独执行，材料保留本地。扫描PDF、多栏阅读顺序、任意长文语义及高并发不是当前已解决问题。
