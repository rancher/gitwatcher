package provider

import (
	"context"
	"net/http"

	"github.com/drone/go-scm/scm"
	webhookv1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
	v1 "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
)

type Provider interface {
	Supports(obj *webhookv1.GitWatcher) bool
	Create(ctx context.Context, obj *webhookv1.GitWatcher) (*webhookv1.GitWatcher, error)
	HandleHook(gitWatchers v1.GitWatcherCache, req *http.Request) (*webhookv1.GitWatcher, scm.Webhook, bool, int, error)
}
