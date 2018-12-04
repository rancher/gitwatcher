package remote

import (
	"errors"

	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/bitbucketcloud"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/bitbucketserver"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/github"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/gitlab"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/model"
)

func New(config interface{}) (model.Remote, error) {
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

	return nil, errors.New("unsupported remote type")
}
