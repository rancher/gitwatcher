package client

const (
	GitWebHookExecutionStatusType           = "gitWebHookExecutionStatus"
	GitWebHookExecutionStatusFieldStatusURL = "statusUrl"
)

type GitWebHookExecutionStatus struct {
	StatusURL string `json:"statusUrl,omitempty" yaml:"statusUrl,omitempty"`
}
