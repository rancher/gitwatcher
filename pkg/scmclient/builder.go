package scmclient

import (
	"errors"

	"github.com/drone/go-scm/scm"
	"github.com/rancher/types/apis/project.cattle.io/v3"
)

func NewClientAuth(config interface{}, credential *v3.SourceCodeCredential) (*scm.Client, error) {
	if config == nil {
		return newDefaultClient()
	}
	switch config := config.(type) {
	case *v3.GithubPipelineConfig:
		return newGithubClientAuth(config, credential)
	case *v3.GitlabPipelineConfig:
		return newGitlabClientAuth(config, credential)
	case *v3.BitbucketCloudPipelineConfig:
		return newBitbucketCloudClientAuth(config, credential)
	case *v3.BitbucketServerPipelineConfig:
		return newBitbucketServerClientAuth(config, credential)
	}
	return nil, errors.New("unsupported provider type")
}

func NewClient(config interface{}) (*scm.Client, error) {
	if config == nil {
		return newDefaultClient()
	}
	switch config := config.(type) {
	case *v3.GithubPipelineConfig:
		return newGithubClient(config)
	case *v3.GitlabPipelineConfig:
		return newGitlabClient(config)
	case *v3.BitbucketCloudPipelineConfig:
		return newBitbucketCloudClient(config)
	case *v3.BitbucketServerPipelineConfig:
		return newBitbucketServerClient(config)
	}
	return nil, errors.New("unsupported provider type")
}
