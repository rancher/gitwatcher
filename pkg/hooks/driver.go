package hooks

import (
	"net/http"

	"github.com/rancher/webhookinator/pkg/hooks/drivers"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
)

var Drivers map[string]Driver

type Driver interface {
	Execute(req *http.Request) (int, error)
}

func RegisterDrivers(client v1.Interface) {
	gitWebHookReceiverLister := client.GitWebHookReceivers("").Controller().Lister()
	gitWebHookExecutions := client.GitWebHookExecutions("")
	Drivers = map[string]Driver{}
	Drivers[drivers.GithubWebhookHeader] = drivers.GithubDriver{
		GitWebHookReceiverLister: gitWebHookReceiverLister,
		GitWebHookExecutions:     gitWebHookExecutions,
	}
	Drivers[drivers.GitlabWebhookHeader] = drivers.GitlabDriver{
		GitWebHookReceiverLister: gitWebHookReceiverLister,
		GitWebHookExecutions:     gitWebHookExecutions,
	}
	Drivers[drivers.BitbucketCloudWebhookHeader] = drivers.BitbucketCloudDriver{
		GitWebHookReceiverLister: gitWebHookReceiverLister,
		GitWebHookExecutions:     gitWebHookExecutions,
	}
	Drivers[drivers.BitbucketServerWebhookHeader] = drivers.BitbucketServerDriver{
		GitWebHookReceiverLister: gitWebHookReceiverLister,
		GitWebHookExecutions:     gitWebHookExecutions,
	}
}
