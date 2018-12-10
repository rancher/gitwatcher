package client

const (
	GitWebHookReceiverStatusType            = "gitWebHookReceiverStatus"
	GitWebHookReceiverStatusFieldConditions = "conditions"
	GitWebHookReceiverStatusFieldHookID     = "hookId"
	GitWebHookReceiverStatusFieldToken      = "token"
)

type GitWebHookReceiverStatus struct {
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	HookID     string      `json:"hookId,omitempty" yaml:"hookId,omitempty"`
	Token      string      `json:"token,omitempty" yaml:"token,omitempty"`
}
