package client

const (
	GitWebHookExecutionStatusType               = "gitWebHookExecutionStatus"
	GitWebHookExecutionStatusFieldAppliedStatus = "appliedStatus"
	GitWebHookExecutionStatusFieldConditions    = "conditions"
	GitWebHookExecutionStatusFieldStatusURL     = "statusUrl"
)

type GitWebHookExecutionStatus struct {
	AppliedStatus string      `json:"appliedStatus,omitempty" yaml:"appliedStatus,omitempty"`
	Conditions    []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	StatusURL     string      `json:"statusUrl,omitempty" yaml:"statusUrl,omitempty"`
}
