package providers

import (
	"errors"

	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/webhookinator/pkg/providers/bitbucketcloud"
	"github.com/rancher/webhookinator/pkg/providers/bitbucketserver"
	"github.com/rancher/webhookinator/pkg/providers/github"
	"github.com/rancher/webhookinator/pkg/providers/gitlab"
	"github.com/rancher/webhookinator/pkg/providers/model"
)

func New(config interface{}) (model.Provider, error) {
	if config == nil {
		return github.New(nil)
	}
	switch config := config.(type) {
	case *v3.GithubPipelineConfig:
		return github.New(config)
	case *v3.GitlabPipelineConfig:
		return gitlab.New(config)
	case *v3.BitbucketCloudPipelineConfig:
		return bitbucketcloud.New(config)
	case *v3.BitbucketServerPipelineConfig:
		return bitbucketserver.New(config)
	}

	return nil, errors.New("unsupported provider type")
}
