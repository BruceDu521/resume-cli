# 单模型评测协议

当前默认单模型，runner显式 --pipeline jev 可选组合。旧评测结果仍属历史版本，不混算。当前重点验证单模型和全文信息提取。

## 比较内容

所有模型使用相同本地 Poppler 文本、JSON 合同、来源范围规则及代码评分。公开extract另用全文+ResumeSchema，不生成评分证据。比较字段遗漏/编造、引用准确性及充分性、要求遗漏/必需优先、逐项状态、报告文字、真实耗时及用量；不能只比较分数或 HTTP 成功。

“整例规则通过”表示一例预设检查全部通过，一项不符即不通过；更高只代表更符合这套政策，不是客观招聘准确率。16 个既有合成案例已用于开发，无独立最终保留集。真实材料只作授权后的有限验证，不作为公开 fixture。

## 执行

`scripts/evaluate.py` 默认 dry-run，不读取 key、不执行 CLI。当前路线为 deepseek、gemini、kimi_code、kimi、openai、mock，默认全部单模型。显式 --pipeline jev 才组合；需要额外注入 TYPESAFE_API_KEY。OpenAI 用户暂缓；不要因为 runner 支持就自动调用。

单供应商继承 `RESUME_AI_API_KEY`。多供应商必须显式提供 `--env-dir`，目录中为 `<provider>.env`，每文件只含该供应商的 `RESUME_AI_API_KEY=...`；不执行 shell，不自动加载项目 dotenv。文件应0600、目录0700、排除Git。Kimi Code key 只用于明确指定的 Code 路线。

```sh
python3 scripts/evaluate.py --routes gemini deepseek
python3 scripts/evaluate.py --routes gemini deepseek --env-dir .local/provider-env \
  --limit 2 --repeats 1 --execute --out .local/eval-new
```

`--suite` 可提供独立清单；`--repeats`控制重复次数。顺序按固定seed打乱，测时不并行启动其他模型评测。原文及规则尽量在看结果前确定；修提示后这些样例必须称回归集，不能继续称独立验证。

## 产物与解释

plan 保存代码/runner/binary/输入hash；每例保存result、stats、stderr、review。失败无最终结果文件，保留失败和费用。runs逐次追加，避免中断后丢失。合成例可用review-evaluation.py辅助检查，但报告文字及证据仍需阅读。

统计包括失败与纠正的端到端耗时。小样本不报稳定p95，不把一次成功称100%可靠。供应商返回usage缺失时费用未知，不当作零；Kimi Code订阅不转换成美元API账单。新版本改变了prompt/schema，应与旧轮分开列出，不能合并成统一延迟或准确率。

## 当前重点

跨行引用修复新增了连续起止块合同与合成离线回归。真实简历需再次检验，但单份材料不能证明多栏/跨页/任意长文稳定。JSON与来源通过也不保证“熟悉/精通”“未体现/不具备”等报告措辞正确。
