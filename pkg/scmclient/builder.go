package scmclient

import (
	"errors"
	"github.com/drone/go-scm/scm/driver/bitbucket"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/driver/stash"
	"github.com/rancher/webhookinator/pkg/utils"

	"github.com/drone/go-scm/scm"
	"github.com/rancher/types/apis/project.cattle.io/v3"
)

func NewClient(config interface{}, credential *v3.SourceCodeCredential) (*scm.Client, error) {
	if config == nil {
		return newDefaultClient()
	}
	switch config := config.(type) {
	case *v3.GithubPipelineConfig:
		return newGithubClient(config, credential)
	case *v3.GitlabPipelineConfig:
		return newGitlabClient(config, credential)
	case *v3.BitbucketCloudPipelineConfig:
		return newBitbucketCloudClient(config, credential)
	case *v3.BitbucketServerPipelineConfig:
		return newBitbucketServerClient(config, credential)
	}
	return nil, errors.New("unsupported provider type")
}

func NewClientBase(providerType string) (*scm.Client, error) {
	switch providerType {
	case utils.GithubType:
		return github.NewDefault(), nil
	case utils.GitlabType:
		return gitlab.NewDefault(), nil
	case utils.BitbucketCloudType:
		return bitbucket.NewDefault(), nil
	case utils.BitbucketServerType:
		return stash.NewDefault(), nil
	default:
		return nil, errors.New("unsupported provider type")
	}
}
