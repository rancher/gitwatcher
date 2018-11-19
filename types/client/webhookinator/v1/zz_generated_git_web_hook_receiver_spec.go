package client

const (
	GitWebHookReceiverSpecType                                = "gitWebHookReceiverSpec"
	GitWebHookReceiverSpecFieldEnabled                        = "enabled"
	GitWebHookReceiverSpecFieldExecutionLabels                = "executionLabels"
	GitWebHookReceiverSpecFieldPR                             = "pr"
	GitWebHookReceiverSpecFieldProvider                       = "provider"
	GitWebHookReceiverSpecFieldPush                           = "push"
	GitWebHookReceiverSpecFieldRepositoryCredentialSecretName = "repositoryCredentialSecretName"
	GitWebHookReceiverSpecFieldRepositoryURL                  = "repositoryUrl"
	GitWebHookReceiverSpecFieldTag                            = "tag"
)

type GitWebHookReceiverSpec struct {
	Enabled                        bool              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	ExecutionLabels                map[string]string `json:"executionLabels,omitempty" yaml:"executionLabels,omitempty"`
	PR                             bool              `json:"pr,omitempty" yaml:"pr,omitempty"`
	Provider                       string            `json:"provider,omitempty" yaml:"provider,omitempty"`
	Push                           bool              `json:"push,omitempty" yaml:"push,omitempty"`
	RepositoryCredentialSecretName string            `json:"repositoryCredentialSecretName,omitempty" yaml:"repositoryCredentialSecretName,omitempty"`
	RepositoryURL                  string            `json:"repositoryUrl,omitempty" yaml:"repositoryUrl,omitempty"`
	Tag                            bool              `json:"tag,omitempty" yaml:"tag,omitempty"`
}
