# 模型厂商与单次评分成本

核对日期：2026-09-21。所有厂商统一使用 RESUME_AI_API_KEY，--model / RESUME_AI_MODEL 可覆盖模型。

| 厂商 | --provider | 默认模型 | 验证状态 |
| --- | --- | --- | --- |
| DeepSeek | deepseek | deepseek-flash | 真实 extract/score 成功 |
| Google | gemini | gemini-3.8-flash | 真实 extract/score 成功 |
| Moonshot | kimi | kimi-k3 | 实测的是 Kimi Code：k3 + https://api.kimi.com/coding/v1；开放平台未成功实测 |
| OpenAI | openai | gpt-6-astra | 严格结构化输出，离线协议/评分流程测试通过，未实调 |
| Anthropic | anthropic / claude | claude-sonnet-5 | 原生 Messages API，离线协议/评分流程测试通过，未实调 |

可选 Jev 是额外匹配器，用 --pipeline jev 启用，另需 TYPESAFE_API_KEY；默认评分不调用它。厂商支持不意味着所有历史模型均支持所用结构化输出参数。

## 实测 token 对应的费用估算

下面来自当前直接评分版本的一次真实简历+JD调用。无重试、无缓存命中；不含 extract、税费、网络、机器或后续人工成本；这是公开 API 费率估算，不是账户账单。

公式：输入 token × 输入单价 / 1,000,000 + 输出 token × 输出单价 / 1,000,000。

| 模型 | 实际输入 / 输出 token | 单次美元 | 同等用量一万次美元 |
| --- | --- | --- | --- |
| DeepSeek Flash | 1,964 / 381 | 峰时0.0010464；谷时0.0005232 | 峰时10.464；谷时5.232 |
| Gemini 3.8 Flash | 2,006 / 332 | 付费档0.0027495 | 27.495 |
| Kimi Code K3 | 2,126 / 309 | 订阅额度，无单次按token账单 | 不能按此估算 |

[DeepSeek 官方价格](https://api-docs.deepseek.com/quick_start/pricing/)：每百万未缓存输入峰时$0.30/谷时$0.15，输出$1.20/$0.60。CLI stats 保守使用峰时价，不自行维护中国节假日日历判断折扣。[Gemini 官方价格](https://ai.google.dev/gemini-api/docs/pricing)：2026年底前输入$0.75、输出$3.75/百万；这里按付费档估算，免费额度不保证可用，账户实际扣费可能不同。该次思考token为0，输出332无需额外推算。

## 未实测厂商：同体量预算示例

仅假设每次2,000输入、400计费输出token（包括适用的思考token），不是它们对这份简历的实测用量。

| 模型 | 每百万输入 / 输出美元 | 单次美元 | 一万次美元 |
| --- | --- | --- | --- |
| GPT-6 Astra | 10 / 50 | 0.040 | 400 |
| Claude Sonnet 5 | 2 / 10 | 0.008 | 80 |

来源：[OpenAI 模型价格](https://developers.openai.com/api/docs/models/gpt-6-astra)、[Anthropic 模型价格](https://platform.claude.com/docs/en/models/overview)。实际用量会因tokenizer、隐藏格式提示、思考和输出长度变化；这些预算不是报价或性能比较。

## 接口依据与测试范围

[OpenAI Chat API](https://developers.openai.com/api/reference/resources/chat)：严格json_schema、拒答/截断处理、token用量；非推理模型不发送reasoning_effort。

[Claude Messages](https://platform.claude.com/docs/en/api/messages/create)及[结构化输出](https://platform.claude.com/docs/en/build-with-claude/structured-outputs)：独立system、x-api-key、anthropic-version、output_config.format、文本内容块和stop_reason。HTTP Schema移除不支持的数值上下界，本地仍执行完整分数范围检查。共享超时、取消、有界重试、HTTPS及响应大小限制保持不变。

离线测试覆盖请求头/端点/格式、正常评分、Schema不被原地修改、拒答/截断/空内容、缓存用量及缺失用量、CLI别名/默认值/统一key。没有使用OpenAI或Claude真实key，没有新增付费模型请求。
