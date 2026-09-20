# 架构设计与实现

2026-09-20：离线实现已完成；Gemini、DeepSeek、Jev 已使用合成样例接通真实 API，最终合成回归比较已完成，结果及限制见 evaluation-results-2026-09-20.md。本文描述当前代码。恢复工作先看 [development.md](development.md)，模型选择见 [evaluation.md](evaluation.md)。

## 流程与边界

parse 只调用本地 Poppler。extract 在解析后生成 Candidate，并投影公开的 Resume 字段。score 使用完整 Candidate 中的工作/项目证据，不能只依赖 extract 的精简字段推断经验。

hybrid：本地 PDF → 简历/JD 并发结构化 → 严格校验 → Jev 批量判断 → 领域评分 → 模板报告。可选 AI 报告只消费确定的 Assessment，不修改分数。

single（baseline 兼容别名）：同一本地原始文本 → 任一所选生成供应商独立完成提取、要求整理、逐项判断与报告 → 相同来源校验和代码评分。不消费 Jev 的结果，也不把对照输出当标准答案。

只保留实际替换点的接口，不使用通用 Agent 框架、数据库或依赖注入容器。

| 模块 | 职责 |
| --- | --- |
| cmd/resume-cli | 信号取消、调用 CLI、退出码 |
| internal/cli | Cobra 参数、配置、依赖组装、输出、用量汇总 |
| internal/app | Parse/Extract/Score 用例；Parser/Structurer/Matcher/Narrator/Baseline 小接口 |
| internal/domain | 数据类型、来源校验、纯函数评分 |
| internal/pdf | 有界本地输入、私有临时文件、可取消 Poppler 子进程 |
| internal/ai | 任务提示词/schema、四家生成适配器、Jev、mock、HTTP、用量 |
| internal/report | 中英文确定性模板 |
| internal/cache | 显式启用的结构化文件缓存 |
| internal/fileio | 有界 UTF-8 读取、原子文件输出 |
| internal/jsonutil | 有限 JSON 修复、严格类型及字段校验 |

## 领域模型

- Document：原文、SHA-256 内容哈希、按页分隔和非空行编号的 Block（ID/page/text）。没有自行推断 PDF 的语义段落。
- Resume：姓名、电话、邮箱、城市、education、skills，对应公开 extract 结构。
- Candidate：Resume 加最多 64 条 Fact。Fact 保存 id、skill/experience/education 类别、block_id、原文 quote。工作/项目目前采用原文证据而非独立结构化时间轴。
- Job：最多 24 条 Requirement，含唯一 ID、类别、必需/优先、JD 原文 span。
- Judgment：requirement_id、status、score、evidence_id、confidence，以及可选 review_reason。
- Assessment：四个分数、not_required、逐项 Finding（要求+判断+证据）、policy_version。
- Report：Assessment 加 comment、interview_questions、language、mock。
- Usage：阶段、供应商/实际模型、输入/缓存/可计费输出 token、时间、attempts、修复标记、用量已知状态及估算成本。

公开事实和引用必须在来源中存在，允许空白差异；引用 ID 必须能回到对应 block。该校验防止虚构原文，不证明语义正确，例如提示注入文字虽然存在于来源中，也不能被当作真实资历；此点由任务指令与评测进一步约束。

Candidate.Ground 在来源验证后将引用扩展为完整来源行，保留否定及上下文；为已提取但尚无证据的 skills/education 字段补充来源行。只复制原文，无法补回模型遗漏的公开字段。该步骤幂等、不修改输入切片，缓存命中也应用。空 facts 拒绝进入评分，避免把提取失败误报为候选人全不匹配。

## Jev 与生成式模型

生成式适配器复用 Generator.Generate，但保持供应商差异：Gemini Interactions 的 steps/model_output、schema、thinking_level；其他三家的 Chat Completions 格式、思考与 usage 映射均独立处理。K3 使用 low reasoning 和严格 JSON Schema；开放平台模型为 kimi-k3，Code 订阅为 k3，端点及账务不同，不自动互换。

Jev 接收 facts 和 requirements，不发送 Resume 的姓名、电话、邮箱字段。每条要求建立两个独立 Choice：

1. satisfied / partial / unmet / unknown，选项包含明确语义条件。
2. 从编号事实中选最佳证据，或 none。

所有问题批量请求；它们不依赖同批其他答案。返回后校验分布完整性、概率范围、近似总和及 choice 一致性；保留 0.01 的舍入容差并加浮点 epsilon，真实 API 两位小数概率可能合计 1.01。非 unknown 必须引用已有事实，unknown 不声称证据。若独立判断声称满足/部分满足/不符，却未选中证据，则保守降为 unknown、0 分、confidence=0，并记录 review_reason=model_judgment_without_evidence；模板显示需复核提示，不隐瞒该冲突。confidence 仅供解释，既非真值概率，也不是候选人的匹配分。

当前不使用 Score 原语：四种有语义的状态已经足以映射首版政策，避免引入无依据的细粒度数值。后续改变等级或权重须更新策略版本与测试。

Jev 本地保守限制：state 48 KiB、总请求 96 KiB；超限报错，不截断证据。中英文合成样例已实测；较长文档、上下文极限和高并发尚未实测。

## 评分与语言

策略 evidence-v1：satisfied=100、partial=50、unmet=0、unknown=0；后两者报告区分。必需项权重 2，优先项 1；按类别求均值，再用技能 50%、经历 35%、教育 15% 加权。中间分数不取整，最终输出四舍五入。

JD 未要求的维度从总分分母中移除；固定数字字段保留 100，同时在 not_required 中标明，不代表实际能力满分。没有可评估要求时报错。基线模型返回的分数也按同一状态映射，不能自行改政策。

默认中文、--lang en 是用户确认的功能。字段名固定英文；事实/引用保留来源语言，评论和面试问题切换语言。重复真实请求仍可能受模型随机性影响，语言无关的政策不代表重新推断一定相同。

暂未实现时间轴合并或精确技能年限计算；提示词禁止从总工龄推断技能年限、禁止重复累加任期。不能在文档中声称已用确定性算法解决此类事实推理。

## 复用与缓存

默认无持久化缓存；--cache-dir 显式开启后，candidate 与 job 分开缓存，键包含阶段版本（当前 candidate:v5、job:v5）、供应商/请求模型/端点、输入内容哈希。阶段版本对应 prompt/schema/纠正合同；修改解析/提示或输出语义必须升级。默认 TTL 24h，防止模型别名长期复用旧结果；缓存未保存实际响应模型作为独立键。

只写通过校验的成功结果，读取再做来源验证；损坏/过期/不兼容视为 miss，I/O 权限错误返回失败。不缓存评分和报告。文件 0600、新建目录 0700；含原文证据，目录排除 Git。若使用已有目录，调用方应确保目录访问权限适合存放简历。

## 失败、取消、文件输出

PDF 20 MiB、文本 160 KiB、JD 64 KiB、AI 响应 2 MiB。PDF 先有界读取，再写入私有临时文件，通过参数数组调用 Poppler，避免 shell 和文件名选项注入。子进程受 context 控制，结束清理临时文件。容器必须有 poppler-data；已用中文 CID 字体样例验证缺包修复。

两个结构化 worker 同时启动；任一失败取消另一个，并等待两个退出再返回，避免遗留请求与漏记用量。总命令默认 90s；HTTP 单次 60s；最多三次针对 429/529/502/503/504 的重试，尊重有界 Retry-After，不重试不明网络错误或 401。JSON/schema、Candidate/Job 来源或 single 整体契约校验失败时，在任务层最多从原始输入重新生成一次；纠正后仍完整校验。端点只接受 HTTPS，禁止重定向。

JSON 修复仅去完整代码围栏/BOM、移除字符串外尾逗号。拒绝重复键、null、深度超过 64、缺失/未知字段、类型错误及多份 JSON。修复不能填补事实。结构化输出/来源校验失败后有一次有上限的纠正生成：复用原始输入，要求符合 schema、引用原文，不发送不可信的前次输出或错误文本。每次调用分别记录 stage（额外调用以 _validation_retry 标记）、耗时与费用；两次仍失败则报错。领域校验不因重试而放宽。

输出文件默认不可覆盖，--force 允许原子替换；禁止覆盖输入及其别名，输出与 stats 不能同路径。采用同目录临时文件和 link/rename。stats 在模型/文件处理失败时也保存；参数、日志配置及已有输出等初始化前错误不保存。

日志不记录 key、整份简历和供应商原始响应。stdout 仅结果；stderr 为 slog 与错误。

## 计量与验证

失败及成功的调用均记录 observed usage。缺失 token 字段标记 unknown；重试不能完整确定费用时 cost_complete=false。Gemini thought tokens 计入账单输出，Chat Completions completion tokens 不重复添加 reasoning tokens。费用使用带日期的美元估算，不能替代账单。

测试使用内存 HTTP 替身，ai/cli 测试额外禁用默认 transport，未实际联网；PDF 测试只运行合成本地输入。覆盖正常流程、无效引用、未知信息、维度权重、JSON 修复、HTTP 重试、取消与 worker 收敛、文件防覆盖和缓存。Docker 禁网跑通三个命令。

已接通 Gemini/DeepSeek/Jev 和 Kimi Code K3 的真实 API，并针对实际返回添加分类规则、概率舍入和证据冲突回归；OpenAI 暂缓，Kimi 开放平台未实测。评测入口见 scripts/evaluate.py，默认只列计划。

## 官方参考

- https://docs.typesafe.ai/api
- https://docs.typesafe.ai/models
- https://ai.google.dev/api/interactions-api
- https://ai.google.dev/gemini-api/docs/thinking
- https://api-docs.deepseek.com/quick_start/pricing/
- https://developers.openai.com/api/docs/models/gpt-6-astra
- https://platform.kimi.ai/
- https://poppler.freedesktop.org/

## 凭据接口

四家生成适配器统一从 `RESUME_AI_API_KEY` 接收密钥，provider/model/base URL 决定请求目标。不读取旧供应商专属变量，也不自动切换密钥或目标。hybrid 同时调用 Jev，使用独立 `TYPESAFE_API_KEY`。CLI 不加载 dotenv，不接受命令行 key；离线测试全部使用合成凭据。
