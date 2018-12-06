package client

import (
	"github.com/rancher/norman/types"
)

const (
	GitWebHookReceiverType                                = "gitWebHookReceiver"
	GitWebHookReceiverFieldAnnotations                    = "annotations"
	GitWebHookReceiverFieldCreated                        = "created"
	GitWebHookReceiverFieldEnabled                        = "enabled"
	GitWebHookReceiverFieldExecutionLabels                = "executionLabels"
	GitWebHookReceiverFieldLabels                         = "labels"
	GitWebHookReceiverFieldName                           = "name"
	GitWebHookReceiverFieldNamespace                      = "namespace"
	GitWebHookReceiverFieldOwnerReferences                = "ownerReferences"
	GitWebHookReceiverFieldPR                             = "pr"
	GitWebHookReceiverFieldProvider                       = "provider"
	GitWebHookReceiverFieldPush                           = "push"
	GitWebHookReceiverFieldRemoved                        = "removed"
	GitWebHookReceiverFieldRepositoryCredentialSecretName = "repositoryCredentialSecretName"
	GitWebHookReceiverFieldRepositoryURL                  = "repositoryUrl"
	GitWebHookReceiverFieldStatus                         = "status"
	GitWebHookReceiverFieldTag                            = "tag"
	GitWebHookReceiverFieldUUID                           = "uuid"
)

type GitWebHookReceiver struct {
	types.Resource
	Annotations                    map[string]string         `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Created                        string                    `json:"created,omitempty" yaml:"created,omitempty"`
	Enabled                        bool                      `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	ExecutionLabels                map[string]string         `json:"executionLabels,omitempty" yaml:"executionLabels,omitempty"`
	Labels                         map[string]string         `json:"labels,omitempty" yaml:"labels,omitempty"`
	Name                           string                    `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace                      string                    `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	OwnerReferences                []OwnerReference          `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
	PR                             bool                      `json:"pr,omitempty" yaml:"pr,omitempty"`
	Provider                       string                    `json:"provider,omitempty" yaml:"provider,omitempty"`
	Push                           bool                      `json:"push,omitempty" yaml:"push,omitempty"`
	Removed                        string                    `json:"removed,omitempty" yaml:"removed,omitempty"`
	RepositoryCredentialSecretName string                    `json:"repositoryCredentialSecretName,omitempty" yaml:"repositoryCredentialSecretName,omitempty"`
	RepositoryURL                  string                    `json:"repositoryUrl,omitempty" yaml:"repositoryUrl,omitempty"`
	Status                         *GitWebHookReceiverStatus `json:"status,omitempty" yaml:"status,omitempty"`
	Tag                            bool                      `json:"tag,omitempty" yaml:"tag,omitempty"`
	UUID                           string                    `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}

type GitWebHookReceiverCollection struct {
	types.Collection
	Data   []GitWebHookReceiver `json:"data,omitempty"`
	client *GitWebHookReceiverClient
}

type GitWebHookReceiverClient struct {
	apiClient *Client
}

type GitWebHookReceiverOperations interface {
	List(opts *types.ListOpts) (*GitWebHookReceiverCollection, error)
	Create(opts *GitWebHookReceiver) (*GitWebHookReceiver, error)
	Update(existing *GitWebHookReceiver, updates interface{}) (*GitWebHookReceiver, error)
	Replace(existing *GitWebHookReceiver) (*GitWebHookReceiver, error)
	ByID(id string) (*GitWebHookReceiver, error)
	Delete(container *GitWebHookReceiver) error
}

func newGitWebHookReceiverClient(apiClient *Client) *GitWebHookReceiverClient {
	return &GitWebHookReceiverClient{
		apiClient: apiClient,
	}
}

func (c *GitWebHookReceiverClient) Create(container *GitWebHookReceiver) (*GitWebHookReceiver, error) {
	resp := &GitWebHookReceiver{}
	err := c.apiClient.Ops.DoCreate(GitWebHookReceiverType, container, resp)
	return resp, err
}

func (c *GitWebHookReceiverClient) Update(existing *GitWebHookReceiver, updates interface{}) (*GitWebHookReceiver, error) {
	resp := &GitWebHookReceiver{}
	err := c.apiClient.Ops.DoUpdate(GitWebHookReceiverType, &existing.Resource, updates, resp)
	return resp, err
}

func (c *GitWebHookReceiverClient) Replace(obj *GitWebHookReceiver) (*GitWebHookReceiver, error) {
	resp := &GitWebHookReceiver{}
	err := c.apiClient.Ops.DoReplace(GitWebHookReceiverType, &obj.Resource, obj, resp)
	return resp, err
}

func (c *GitWebHookReceiverClient) List(opts *types.ListOpts) (*GitWebHookReceiverCollection, error) {
	resp := &GitWebHookReceiverCollection{}
	err := c.apiClient.Ops.DoList(GitWebHookReceiverType, opts, resp)
	resp.client = c
	return resp, err
}

func (cc *GitWebHookReceiverCollection) Next() (*GitWebHookReceiverCollection, error) {
	if cc != nil && cc.Pagination != nil && cc.Pagination.Next != "" {
		resp := &GitWebHookReceiverCollection{}
		err := cc.client.apiClient.Ops.DoNext(cc.Pagination.Next, resp)
		resp.client = cc.client
		return resp, err
	}
	return nil, nil
}

func (c *GitWebHookReceiverClient) ByID(id string) (*GitWebHookReceiver, error) {
	resp := &GitWebHookReceiver{}
	err := c.apiClient.Ops.DoByID(GitWebHookReceiverType, id, resp)
	return resp, err
}

func (c *GitWebHookReceiverClient) Delete(container *GitWebHookReceiver) error {
	return c.apiClient.Ops.DoResourceDelete(GitWebHookReceiverType, &container.Resource)
}
