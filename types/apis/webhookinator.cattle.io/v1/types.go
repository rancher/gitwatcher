package v1

import (
	"github.com/rancher/norman/types"
	"github.com/rancher/norman/types/factory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	APIVersion = types.APIVersion{
		Group:   "webhookinator.cattle.io",
		Version: "v1",
		Path:    "/v1-webhook",
	}
	Schemas = factory.
		Schemas(&APIVersion).
		MustImport(&APIVersion, GitWebHookReceiver{}).
		MustImport(&APIVersion, GitWebHookExecution{})
)

type GitWebHookReceiver struct {
	types.Namespaced

	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GitWebHookReceiverSpec `json:"spec"`
}

type GitWebHookReceiverSpec struct {
	RepositoryURL                  string            `json:"repositoryUrl,omitempty"`
	RepositoryCredentialSecretName string            `json:"repositoryCredentialSecretName,omitempty"`
	Provider                       string            `json:"provider,omitempty"`
	Push                           bool              `json:"push,omitempty"`
	PR                             bool              `json:"pr,omitempty"`
	Tag                            bool              `json:"tag,omitempty"`
	ExecutionLabels                map[string]string `json:"executionLabels,omitempty"`
	Enabled                        bool              `json:"enabled,omitempty"`
}

type GitWebHookExecution struct {
	types.Namespaced

	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitWebHookExecutionSpec   `json:"spec,omitempty"`
	Status GitWebHookExecutionStatus `json:"status,omitempty"`
}

type GitWebHookExecutionSpec struct {
	Payload                string `json:"payload,omitempty"`
	GitWebHookReceiverName string `json:"gitWebHookReceiverName,omitempty"`
	Commit                 string `json:"commit,omitempty"`
	Branch                 string `json:"branch,omitempty"`
	Tag                    string `json:"tag,omitempty"`
	PR                     string `json:"pr,omitempty"`
	SourceLink             string `json:"sourceLink,omitempty"`
}

type GitWebHookExecutionStatus struct {
	StatusURL string `json:"statusUrl,omitempty"`
}
