package client

import (
	"github.com/rancher/norman/types"
)

const (
	GitWebHookExecutionType                        = "gitWebHookExecution"
	GitWebHookExecutionFieldAnnotations            = "annotations"
	GitWebHookExecutionFieldAuthor                 = "author"
	GitWebHookExecutionFieldAuthorAvatar           = "authorAvatar"
	GitWebHookExecutionFieldAuthorEmail            = "authorEmail"
	GitWebHookExecutionFieldBranch                 = "branch"
	GitWebHookExecutionFieldCommit                 = "commit"
	GitWebHookExecutionFieldCreated                = "created"
	GitWebHookExecutionFieldGitWebHookReceiverName = "gitWebHookReceiverName"
	GitWebHookExecutionFieldLabels                 = "labels"
	GitWebHookExecutionFieldMessage                = "message"
	GitWebHookExecutionFieldName                   = "name"
	GitWebHookExecutionFieldNamespace              = "namespace"
	GitWebHookExecutionFieldOwnerReferences        = "ownerReferences"
	GitWebHookExecutionFieldPR                     = "pr"
	GitWebHookExecutionFieldPayload                = "payload"
	GitWebHookExecutionFieldRemoved                = "removed"
	GitWebHookExecutionFieldRepositoryURL          = "repositoryUrl"
	GitWebHookExecutionFieldSourceLink             = "sourceLink"
	GitWebHookExecutionFieldStatus                 = "status"
	GitWebHookExecutionFieldTag                    = "tag"
	GitWebHookExecutionFieldTitle                  = "title"
	GitWebHookExecutionFieldUUID                   = "uuid"
)

type GitWebHookExecution struct {
	types.Resource
	Annotations            map[string]string          `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Author                 string                     `json:"author,omitempty" yaml:"author,omitempty"`
	AuthorAvatar           string                     `json:"authorAvatar,omitempty" yaml:"authorAvatar,omitempty"`
	AuthorEmail            string                     `json:"authorEmail,omitempty" yaml:"authorEmail,omitempty"`
	Branch                 string                     `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit                 string                     `json:"commit,omitempty" yaml:"commit,omitempty"`
	Created                string                     `json:"created,omitempty" yaml:"created,omitempty"`
	GitWebHookReceiverName string                     `json:"gitWebHookReceiverName,omitempty" yaml:"gitWebHookReceiverName,omitempty"`
	Labels                 map[string]string          `json:"labels,omitempty" yaml:"labels,omitempty"`
	Message                string                     `json:"message,omitempty" yaml:"message,omitempty"`
	Name                   string                     `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace              string                     `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	OwnerReferences        []OwnerReference           `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
	PR                     string                     `json:"pr,omitempty" yaml:"pr,omitempty"`
	Payload                string                     `json:"payload,omitempty" yaml:"payload,omitempty"`
	Removed                string                     `json:"removed,omitempty" yaml:"removed,omitempty"`
	RepositoryURL          string                     `json:"repositoryUrl,omitempty" yaml:"repositoryUrl,omitempty"`
	SourceLink             string                     `json:"sourceLink,omitempty" yaml:"sourceLink,omitempty"`
	Status                 *GitWebHookExecutionStatus `json:"status,omitempty" yaml:"status,omitempty"`
	Tag                    string                     `json:"tag,omitempty" yaml:"tag,omitempty"`
	Title                  string                     `json:"title,omitempty" yaml:"title,omitempty"`
	UUID                   string                     `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}

type GitWebHookExecutionCollection struct {
	types.Collection
	Data   []GitWebHookExecution `json:"data,omitempty"`
	client *GitWebHookExecutionClient
}

type GitWebHookExecutionClient struct {
	apiClient *Client
}

type GitWebHookExecutionOperations interface {
	List(opts *types.ListOpts) (*GitWebHookExecutionCollection, error)
	Create(opts *GitWebHookExecution) (*GitWebHookExecution, error)
	Update(existing *GitWebHookExecution, updates interface{}) (*GitWebHookExecution, error)
	Replace(existing *GitWebHookExecution) (*GitWebHookExecution, error)
	ByID(id string) (*GitWebHookExecution, error)
	Delete(container *GitWebHookExecution) error
}

func newGitWebHookExecutionClient(apiClient *Client) *GitWebHookExecutionClient {
	return &GitWebHookExecutionClient{
		apiClient: apiClient,
	}
}

func (c *GitWebHookExecutionClient) Create(container *GitWebHookExecution) (*GitWebHookExecution, error) {
	resp := &GitWebHookExecution{}
	err := c.apiClient.Ops.DoCreate(GitWebHookExecutionType, container, resp)
	return resp, err
}

func (c *GitWebHookExecutionClient) Update(existing *GitWebHookExecution, updates interface{}) (*GitWebHookExecution, error) {
	resp := &GitWebHookExecution{}
	err := c.apiClient.Ops.DoUpdate(GitWebHookExecutionType, &existing.Resource, updates, resp)
	return resp, err
}

func (c *GitWebHookExecutionClient) Replace(obj *GitWebHookExecution) (*GitWebHookExecution, error) {
	resp := &GitWebHookExecution{}
	err := c.apiClient.Ops.DoReplace(GitWebHookExecutionType, &obj.Resource, obj, resp)
	return resp, err
}

func (c *GitWebHookExecutionClient) List(opts *types.ListOpts) (*GitWebHookExecutionCollection, error) {
	resp := &GitWebHookExecutionCollection{}
	err := c.apiClient.Ops.DoList(GitWebHookExecutionType, opts, resp)
	resp.client = c
	return resp, err
}

func (cc *GitWebHookExecutionCollection) Next() (*GitWebHookExecutionCollection, error) {
	if cc != nil && cc.Pagination != nil && cc.Pagination.Next != "" {
		resp := &GitWebHookExecutionCollection{}
		err := cc.client.apiClient.Ops.DoNext(cc.Pagination.Next, resp)
		resp.client = cc.client
		return resp, err
	}
	return nil, nil
}

func (c *GitWebHookExecutionClient) ByID(id string) (*GitWebHookExecution, error) {
	resp := &GitWebHookExecution{}
	err := c.apiClient.Ops.DoByID(GitWebHookExecutionType, id, resp)
	return resp, err
}

func (c *GitWebHookExecutionClient) Delete(container *GitWebHookExecution) error {
	return c.apiClient.Ops.DoResourceDelete(GitWebHookExecutionType, &container.Resource)
}
