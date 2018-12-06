package client

const (
	GitWebHookReceiverStatusType            = "gitWebHookReceiverStatus"
	GitWebHookReceiverStatusFieldConditions = "conditions"
	GitWebHookReceiverStatusFieldToken      = "token"
)

type GitWebHookReceiverStatus struct {
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Token      string      `json:"token,omitempty" yaml:"token,omitempty"`
}
