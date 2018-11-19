package client

const (
	GitWebHookExecutionSpecType                        = "gitWebHookExecutionSpec"
	GitWebHookExecutionSpecFieldBranch                 = "branch"
	GitWebHookExecutionSpecFieldCommit                 = "commit"
	GitWebHookExecutionSpecFieldGitWebHookReceiverName = "gitWebHookReceiverName"
	GitWebHookExecutionSpecFieldPR                     = "pr"
	GitWebHookExecutionSpecFieldPayload                = "payload"
	GitWebHookExecutionSpecFieldSourceLink             = "sourceLink"
	GitWebHookExecutionSpecFieldTag                    = "tag"
)

type GitWebHookExecutionSpec struct {
	Branch                 string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit                 string `json:"commit,omitempty" yaml:"commit,omitempty"`
	GitWebHookReceiverName string `json:"gitWebHookReceiverName,omitempty" yaml:"gitWebHookReceiverName,omitempty"`
	PR                     string `json:"pr,omitempty" yaml:"pr,omitempty"`
	Payload                string `json:"payload,omitempty" yaml:"payload,omitempty"`
	SourceLink             string `json:"sourceLink,omitempty" yaml:"sourceLink,omitempty"`
	Tag                    string `json:"tag,omitempty" yaml:"tag,omitempty"`
}
