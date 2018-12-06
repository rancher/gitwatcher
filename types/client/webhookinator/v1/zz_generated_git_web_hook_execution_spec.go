package client

const (
	GitWebHookExecutionSpecType                        = "gitWebHookExecutionSpec"
	GitWebHookExecutionSpecFieldAuthor                 = "author"
	GitWebHookExecutionSpecFieldAuthorAvatar           = "authorAvatar"
	GitWebHookExecutionSpecFieldAuthorEmail            = "authorEmail"
	GitWebHookExecutionSpecFieldBranch                 = "branch"
	GitWebHookExecutionSpecFieldCommit                 = "commit"
	GitWebHookExecutionSpecFieldGitWebHookReceiverName = "gitWebHookReceiverName"
	GitWebHookExecutionSpecFieldMessage                = "message"
	GitWebHookExecutionSpecFieldPR                     = "pr"
	GitWebHookExecutionSpecFieldPayload                = "payload"
	GitWebHookExecutionSpecFieldRepositoryURL          = "repositoryUrl"
	GitWebHookExecutionSpecFieldSourceLink             = "sourceLink"
	GitWebHookExecutionSpecFieldTag                    = "tag"
	GitWebHookExecutionSpecFieldTitle                  = "title"
)

type GitWebHookExecutionSpec struct {
	Author                 string `json:"author,omitempty" yaml:"author,omitempty"`
	AuthorAvatar           string `json:"authorAvatar,omitempty" yaml:"authorAvatar,omitempty"`
	AuthorEmail            string `json:"authorEmail,omitempty" yaml:"authorEmail,omitempty"`
	Branch                 string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit                 string `json:"commit,omitempty" yaml:"commit,omitempty"`
	GitWebHookReceiverName string `json:"gitWebHookReceiverName,omitempty" yaml:"gitWebHookReceiverName,omitempty"`
	Message                string `json:"message,omitempty" yaml:"message,omitempty"`
	PR                     string `json:"pr,omitempty" yaml:"pr,omitempty"`
	Payload                string `json:"payload,omitempty" yaml:"payload,omitempty"`
	RepositoryURL          string `json:"repositoryUrl,omitempty" yaml:"repositoryUrl,omitempty"`
	SourceLink             string `json:"sourceLink,omitempty" yaml:"sourceLink,omitempty"`
	Tag                    string `json:"tag,omitempty" yaml:"tag,omitempty"`
	Title                  string `json:"title,omitempty" yaml:"title,omitempty"`
}
