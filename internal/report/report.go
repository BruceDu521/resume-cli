package report

import (
	"fmt"
	"strings"

	"resume-cli/internal/domain"
)

type Result struct {
	domain.Assessment
	Comment   string   `json:"comment"`
	Questions []string `json:"interview_questions"`
	Language  string   `json:"language"`
	Mock      bool     `json:"mock"`
}

func Render(a domain.Assessment, lang string, mock bool) Result {
	comments, questions := []string{}, []string{}
	for _, f := range a.Findings {
		r, s := f.Requirement.Text, f.Judgment.Status
		var comment, question string
		if lang == "en" {
			switch s {
			case "satisfied":
				comment = fmt.Sprintf("Evidence supports [%s].", r)
				question = fmt.Sprintf("Describe your specific responsibilities and outcomes relevant to [%s].", r)
			case "partial":
				comment = fmt.Sprintf("Evidence partially supports [%s]; scope and depth need confirmation.", r)
				question = fmt.Sprintf("Which parts of [%s] have you personally handled, and in what environment?", r)
			case "unmet":
				comment = fmt.Sprintf("The stated evidence conflicts with [%s]; clarify during the interview.", r)
				question = fmt.Sprintf("Your resume indicates a gap against [%s]. Has your experience changed?", r)
			default:
				comment = fmt.Sprintf("The resume does not establish [%s]; this is missing evidence, not proof of inability.", r)
				question = fmt.Sprintf("Do you have experience relevant to [%s]? Please provide an example.", r)
			}
		} else {
			switch s {
			case "satisfied":
				comment = fmt.Sprintf("已有证据支持「%s」。", r)
				question = fmt.Sprintf("请结合「%s」介绍你实际承担的工作和结果。", r)
			case "partial":
				comment = fmt.Sprintf("「%s」有部分相关证据，责任范围和实践深度需要确认。", r)
				question = fmt.Sprintf("围绕「%s」，哪些部分由你实际负责？当时是什么环境？", r)
			case "unmet":
				comment = fmt.Sprintf("原文证据与「%s」存在不符，需要面试确认。", r)
				question = fmt.Sprintf("简历体现的经历与「%s」有差距，最近是否有补充经验？", r)
			default:
				comment = fmt.Sprintf("简历尚未体现「%s」，不代表候选人不具备该能力。", r)
				question = fmt.Sprintf("你是否有「%s」相关经历？请给出具体例子。", r)
			}
		}
		comments = append(comments, comment)
		if s != "satisfied" {
			questions = append(questions, question)
		}
	}
	if len(questions) == 0 && len(a.Findings) > 0 {
		r := a.Findings[0].Requirement.Text
		if lang == "en" {
			questions = append(questions, fmt.Sprintf("Describe a project demonstrating [%s], your decisions and results.", r))
		} else {
			questions = append(questions, fmt.Sprintf("请介绍能体现「%s」的项目、你的决策及结果。", r))
		}
	}
	if len(questions) > 3 {
		questions = questions[:3]
	}
	if len(a.NotRequired) > 0 {
		if lang == "en" {
			comments = append(comments, "Dimensions without JD requirements display 100 as a placeholder and are excluded from the overall score.")
		} else {
			comments = append(comments, "JD 未要求的维度以 100 占位，不计入总分。")
		}
	}
	return Result{Assessment: a, Comment: strings.Join(comments, " "), Questions: questions, Language: lang, Mock: mock}
}
