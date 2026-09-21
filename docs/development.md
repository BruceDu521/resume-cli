# 开发恢复说明

## 当前决定（2026-09-21）

用户明确放弃 Jev，只保留单模型以减少复杂度。不要恢复多模型编排或两个key。三个命令、全部五项增强、中英文及广覆盖离线测试仍是核心要求。

score 直接由 Evaluator 一次生成结构、判断和报告，校验失败最多纠正一次；本地计算分数。移除了 Jev adapter、Matcher/Narrator/Baseline 分支、pipeline/jev-model/report参数及相关env。extract可单独缓存，score不支持cache-dir。

真实Gemini单模型诊断发现跨行quote只标单行ID，而旧合同要求单行包含。Poppler原文没有因此丢字，主要是应用证据结构与prompt不协调。已添加 end_block_id、最多16个连续来源块、严格范围原文校验及上下文保留；候选缓存candidate:v6。错误信息区分JSON与固定领域校验原因，不输出原文。未放宽为全文任意匹配。

## 测试和凭据

Go1.25.5；本机 Poppler26.04.0；Docker OrbStack Debian Poppler22.12.0及poppler-data。模块缓存/private/tmp/resume-cli-gomod，构建缓存/private/tmp/resume-cli-gobuild，GOPROXY=off。

用户授权全包离线测试；本机hook要求命令以CLAUDE_APPROVED=1开头。AI/CLI transport默认禁网；禁止测试读取.env或数据库。

用户已授权本项目DeepSeek、Gemini及KimiCode K3真实测试，OpenAI暂缓。只读本任务提供的key；RESUME_AI_API_KEY为统一接口。各厂商私有key在.local/provider-env/*.env；主.env当前DS。旧Jev配置已转存.local/retired-jev-config.env，不加载、不调用、不提交。

真实简历路径见本地 session notes；只供测试，不修改该PDF，不提交原文或结果。结果.local/real-resume/，上一轮score-v1为旧程序，single-only-v1为本轮。见最新实测报告，不把旧失败抹掉，也不拿旧调试轮当独立准确率。

## 剩余工作

- 继续关注字段完整性、跨行证据语义充分性、多栏/跨页和自由评论；引用真实不等于判断正确。
- 公开实现仓库、演示视频及招聘提交未完成，未建remote/push。
- OpenAI、Windows、高并发、OCR、确定性任职区间计算未实现或验证。
- 保留旧评测报告并标历史，当前使用方式以README和architecture.md为准。

## 本轮验证结果

Go全包race/vet通过，总覆盖87.7%，领域95.9%，AI86.0%。Python4项凭据测试和新runner mock2例通过。真实v1 Gem成功7.455s、DS/Kimi分别字段来源/JD数量失败；v2新增具体安全纠错反馈后DS首次成功7.592s、Kimi纠正后成功55.471s。不要称DS是反馈修好，首次成功也可能是随机波动。三者成功报告83分、跨行引用实际有效。详细见evaluation-single-only-2026-09-21.md；所有评测进程已结束，不需重复调用。

Dockerfile本轮build因Docker Hub auth EOF失败，不能写成构建通过；在旧resume-cli:local镜像挂载当前linux/arm64二进制，三个命令禁网验证已通过。
