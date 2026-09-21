package report

type Result struct {
	Evaluation
	Policy   string `json:"policy_version"`
	Language string `json:"language"`
	Mock     bool   `json:"mock"`
}
