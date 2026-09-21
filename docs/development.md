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
