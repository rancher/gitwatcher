package webhook

import (
	"context"
	"time"

	"github.com/rancher/gitwatcher/pkg/provider"
	"github.com/rancher/gitwatcher/pkg/provider/github"
	"github.com/rancher/gitwatcher/pkg/provider/polling"

	webhookv1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
	webhookcontrollerv1 "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
	webhookv1controller "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
	"github.com/rancher/gitwatcher/pkg/types"
	"github.com/rancher/wrangler/pkg/ticker"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	refreshInterval = 30
)

func Register(ctx context.Context, rContext *types.Context) error {
	secretsLister := rContext.Core.Core().V1().Secret().Cache()

	wh := webhookHandler{
		ctx:             ctx,
		gitWatcherCache: rContext.Webhook.Gitwatcher().V1().GitWatcher().Cache(),
		gitWatcher:      rContext.Webhook.Gitwatcher().V1().GitWatcher(),
	}

	apply := rContext.Apply.WithCacheTypes(
		rContext.Webhook.Gitwatcher().V1().GitWatcher(),
		rContext.Webhook.Gitwatcher().V1().GitCommit())
	wh.providers = append(wh.providers, github.NewGitHub(secretsLister, apply))
	wh.providers = append(wh.providers, polling.NewPolling(secretsLister, apply))

	rContext.Webhook.Gitwatcher().V1().GitWatcher().OnChange(ctx, "webhook-receiver",
		webhookv1controller.UpdateGitWatcherOnChange(rContext.Webhook.Gitwatcher().V1().GitWatcher().Updater(), wh.onChange))

	wh.start()
	return nil
}

type webhookHandler struct {
	ctx             context.Context
	gitWatcher      webhookcontrollerv1.GitWatcherController
	gitWatcherCache webhookcontrollerv1.GitWatcherCache
	providers       []provider.Provider
}

func (w *webhookHandler) onChange(key string, obj *webhookv1.GitWatcher) (*webhookv1.GitWatcher, error) {
	if obj == nil {
		return nil, nil
	}

	for _, provider := range w.providers {
		if provider.Supports(obj) {
			return provider.Create(w.ctx, obj)
		}
	}

	return obj, nil
}

func (w *webhookHandler) start() {
	go func() {
		for range ticker.Context(w.ctx, refreshInterval*time.Second) {
			modules, err := w.gitWatcherCache.List("", labels.NewSelector())
			if err == nil {
				for _, m := range modules {
					if m.Status.HookID == "" {
						w.gitWatcher.Enqueue(m.Namespace, m.Name)
					}
				}
			}
		}
	}()
}
