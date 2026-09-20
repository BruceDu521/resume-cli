# 单模型与 Jev 组合比较

用户追加问题：是否可以只配置一家模型，其成本、质量和速度与组合流程有何差别？本轮新增 single 模式，并实测 DeepSeek、Gemini 与 Kimi Code K3。OpenAI 按用户要求暂缓。

## 配置与执行模式

原题允许大模型 API，没有指定模型数量、供应商或 key 传递方式；要求 README 解释环境变量配置。现有接口扩展为：

| 配置 | 环境变量 | 参数覆盖 |
| --- | --- | --- |
| 供应商 | RESUME_AI_PROVIDER | --provider |
| 生成模型 ID | RESUME_AI_MODEL | --model |
| 评分模式 | RESUME_AI_PIPELINE | --pipeline single/hybrid |
| 生成 API 地址 | RESUME_AI_BASE_URL | --base-url |
| Jev 模型 | RESUME_JEV_MODEL | --jev-model |
| API key | 对应供应商的 *_API_KEY | 不支持命令行明文 key |

`parse` 不调用 AI；`extract` 只需要所选供应商 key。`score --pipeline single` 通常一次生成候选事实、JD 要求、逐项判断及文字报告，然后执行来源验证和统一代码评分；校验失败最多纠正一次。`baseline` 保留为 single 的兼容别名。

`hybrid` 使用所选供应商并行整理简历和 JD，再由 Jev 判断；默认模板报告，可选择 AI 报告。它支持分开缓存简历/JD。single 不使用该结构化缓存，也不依赖 Jev key。两种模式共用文件读取、JSON 校验、领域评分及输出，未复制一套业务流程。

当前保留 hybrid 默认值，provider 仍需显式选择。只有一个 key 时直接指定 single，不自动跨供应商降级。

## 测试方法

- 16 个相同合成案例，DS/Gemini 四路径各重复两次，随机交错，共 128 次；Kimi Code K3 稍后单独各跑一次，共 16 次。
- 报告中文或英文按相同案例指定；所有 key 只读取本项目环境文件，没有真实个人简历上传。
- 固定模型与配置：DeepSeek Flash 禁用 thinking，Gemini 3.8 Flash 为 low，Jev 1.13.0；K3 为 low，严格 JSON Schema。
- single 自带生成式报告，hybrid 默认本地模板，因此二者完成同一公开功能但生成 token 工作量不同。
- 关闭应用缓存复用，供应商缓存按 usage 记录；耗时包含重试、纠正和失败。比较的是完整 JSON 输出时间，不是首 token 时间。
- 当前样例都是开发回归集。hybrid 先前经历更多针对这些样例的调试，不能据此宣称某种模型或架构天然更准确；本轮未中途调整判断提示词。
- “规则一致”指一次试验的要求类别、必需/优先、状态、证据及基本报告检查全部通过。规则来自合成输入，差异逐项阅读核对，不是独立人工标注准确率。

## 结果

| 路径 | 成功输出 | 完整规则一致 | 中位耗时（含失败） | 范围 | 每次尝试平均估算费用 |
| --- | ---: | ---: | ---: | ---: | ---: |
| DeepSeek 单模型 | 31/32 | 20/32 | 2.614 s | 1.900–5.798 s | $0.000795 |
| DeepSeek + Jev | 32/32 | 30/32 | 2.607 s | 1.987–3.089 s | $0.000621 |
| Gemini 单模型 | 32/32 | 27/32 | 3.609 s | 2.999–4.634 s | $0.002931 |
| Gemini + Jev | 32/32 | 32/32 | 3.848 s | 3.336–6.423 s | $0.002536 |
| Kimi Code K3 单模型 | 15/16 | 15/16 | 23.809 s | 10.318–61.208 s | 订阅额度，未知美元费用 |

费用为已观察 API usage 的公开价估算，含纠正及失败调用；DeepSeek 使用保守高峰价，并非实际账单。Kimi Code 消耗订阅额度，不显示伪造的按请求美元费用。输入很短，不可直接推算真实长简历、高并发或生产 p95。

DS 单模型没有减少中位耗时，Gemini 单模型略快；两者平均估算费用均高于对应 hybrid。原因是生成模型要输出逐项判断和完整报告，输出 token 增加，而 Jev 的少量输入费用较低。DS 生成模型每次尝试平均输出约 576 tokens（single）与 359（hybrid），Gemini 为 659 与 533；hybrid 的额外 Jev 平均估算费用分别约 $0.000082 和 $0.000093。

## 内容差异

- DS + Jev：一例将全部硬要求标为优先项，另一例把通用 Rust 开发要求归为 experience；32 次匹配状态符合当前规则。该结果也说明上一轮通过不保证下轮一致。
- Gemini + Jev：32 次完整规则通过，但结构化中间结果仍有一次 skills 数组为空，工作证据尚在。评分通过不等于提取完整。
- DS single：一次经过纠正仍未通过严格校验；另有明确未体现 Rust 却判断 unmet、已知基础技能低估为 partial、硬要求/优先混淆及类别差异。某些复合运维要求被判 partial，而约定规则依据明确责任否定判 unmet。
- Gemini single：四次对明确生产责任缺口判 partial，约定为 unmet；另一次把岗位标题额外拆成技能要求。后者本例没有改变总分，但仍可能影响混合要求的权重。
- Kimi Code K3：15/16 成功且完整规则通过，另一个缺失 Rust 案例在约 60 秒单请求上限时失败，没有可评价的内容。三次成功任务各经过一次校验纠正，其中最长的总耗时约 61 秒。不能把超时样例算作正确，也不能推断其内容会错。

这里的 partial/unmet 是本工具明确选择的证据政策，并非客观唯一的招聘标准。基于部署经历给运维要求部分分也有解释空间，故报告列出具体差异，不能把全部分歧描述成事实捏造。DS 将缺失 Rust 直接判“不符”则违反了“缺少证据不代表没有能力”的约定。

## Kimi 的接口区别

用户提供的是 **Kimi Code 订阅 key**，不是开放平台 key；起初在两个开放平台区域各一次 401，未混入有效模型测评。确认产品后按官方文档使用 `https://api.kimi.com/coding/v1`、模型 `k3`，成功响应也返回 `k3`，未使用会动态变化的 `kimi-for-coding` 代替。

Code 的质量/速度仅代表本次订阅接口与账号配置；其订阅额度不能按开放平台单价宣称为实际费用。开放平台 K3 适配器支持 `kimi-k3`，本次未取得该产品 key，未验证；其现有 USD 估算不包括独立缓存写入费用，明确标记不完整。

参考：[Code 模型与端点](https://www.kimi.com/code/docs/en/kimi-code/models.html)、[K3 推理档位](https://platform.kimi.com/docs/guide/use-reasoning-effort)、[结构化输出](https://platform.kimi.com/docs/guide/response_format)、[开放平台计费](https://platform.kimi.ai/docs/pricing/chat)。

K3 的 API 已返回并可观察到输入 28,911、输出 11,898 tokens，合计 40,809；超时请求没有返回 usage，因此这些不是完整订阅扣额。首次冒烟和误用开放平台产生的两次 401 不计入这批 16 次。K3 成功样例的中位耗时与含失败的统计分别保留原始数据，可复核，未用快失败美化结果。

## 当前建议

保留两种模式：单 key 用户可直接运行 single；本次样例中需要更稳定的逐项判断、重复解析或换 JD 处理时，优先 hybrid。DS + Jev 的速度/费用更有优势，Gemini + Jev 本轮要求整理更一致。K3 作为较慢的独立对照提供额外信息，是否值得其等待时间取决于使用场景。

完整汇总见 [JSON](evaluation-single-vs-hybrid-2026-09-20.json)。原始输入哈希、请求用量、错误和结果分别保留在本机 `.local/eval-single-comparison-v1`、`.local/eval-kimi-code-v1`；没有隐藏失败或将订阅费用算作零。后续需要新的独立、接近真实长度的保留集，再评估泛化效果。
