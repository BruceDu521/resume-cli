# 开发恢复说明

## 当前约定（2026-09-21，优先于早期记录）

默认 single 单模型，保留显式 `--pipeline jev`，`hybrid` 为兼容别名。用户只是要求先把单模型做好，并未要求删除 Jev；上一轮删除已纠正。不要擅自删选项，也不要在默认路径偷偷调用 Jev。

用户质疑了公共提取任务中的编号块与64条事实。当前 public extract 独立用完整 d.Text + ResumeSchema，只输出 name/phone/email/city/education/skills，依据工作及项目实际描述整理技能，不生成facts/block_id。只校验结构与空值约定，不将技能名称逐字匹配原文当作语义正确性。独立Extractor接口及resume:v1缓存，避免复用旧Candidate缓存。

评分内部保留来源可追溯，b1/b2是代码给Poppler输出非空行添加的序号，不是PDF段落，模型原先收到全部行而非某64行。64条事实是先前未经充分验证的工程取值，现已从prompt和领域验证中移除；保留总字节及Jev请求预算。Jev内部candidate:v8/job:v6缓存；当前64KiB JD上限与64条事实是不同概念。

score single只生成matches（每项带引用数组）/comment/interview_questions，代码生成内部ID，不提取个人字段。必要时最多纠正一次。Jev路线独立Candidate/Job并行后Match，保留取消及等待worker退出；报告本地模板，无额外AI报告调用。EndBlockID跨行原文范围保留，不限制行数；不再限制24项JD要求。默认single支持一项要求多处不连续引用，可选Jev保留旧单证据选择。

## 测试与配置

Go1.25.5、macOS Poppler26.04.0；依赖缓存/private/tmp/resume-cli-gomod与/private/tmp/resume-cli-gobuild，GOPROXY=off。本机测试hook要求CLAUDE_APPROVED=1开头；用户授权全包离线测试。AI/CLI默认transport禁网，测试不能读取.env或任何数据库。

RESUME_AI_API_KEY为所选生成供应商key，主.env当前DS且RESUME_AI_PIPELINE=single。任务keys在.local/provider-env各0600文件；可选Jev key已从本任务私有备份恢复到主.env，仅显式Jev评分才读取。不得发现其他项目凭据。OpenAI暂缓；Kimi使用Code订阅k3及api.kimi.com/coding/v1，不混开放平台。

真实简历路径在.local/session-notes.md，用户授权测试但不修改该PDF、不提交原文/个人资料。最新全文提取输出.local/real-resume/public-extract-v1/：DS 2.477s、Gemini3.285s，均一次成功，均保留14项明确技能及个人/学历字段。DS归纳43项（技术与工作能力混合），Gemini29项；数量不代表优劣。Gemini把原文Django REST API具体化为Django REST framework，原文不能确认该框架，需记录为待改进的技能归纳，不假称全字段准确。没有为此继续调prompt/重复实测。

旧score-v1、single-only-v1/v2是不同程序和提示版本，保留失败，不混算统计。新的可选Jev只跑离线替身回归，本轮没有实调Jev/Kimi/OpenAI。

## 尚未完成

- 技能归纳粒度、遗漏及无依据具体化，复杂版面/跨页来源、评分评论语义仍需检查。
- Dockerfile最近重建因Docker Hub auth EOF未完成；上一轮已有镜像挂载当时新Linux二进制禁网运行通过，不代表当前版本重新构建成功。
- 公开GitHub仓库、演示视频、招聘提交未完成；无remote/push。
- Windows、OpenAI、高并发、OCR、确定性技能年限/任职区间未验证或实现。

## 本轮 review 收尾

详见 [修复及限制](review-fixes-2026-09-21.md)。全包离线 race 单测和 vet 通过；仅执行一次 Gemini 真实评分，无纠正重试，模型调用6.912秒、CLI6.958秒、评测脚本墙钟9.235秒。5项要求均保留，实际输出多处引用；评论仍无依据写了“全日制”，不把流程成功视为语义准确。结果位于.local/real-resume/review-single-v1/，无后台进程，不追加批量评测。
