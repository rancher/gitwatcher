package provider

import (
	"context"
	"net/http"

	webhookv1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
)

type Provider interface {
	Supports(obj *webhookv1.GitWatcher) bool
	Create(ctx context.Context, obj *webhookv1.GitWatcher) (*webhookv1.GitWatcher, error)
	HandleHook(ctx context.Context, req *http.Request) (int, error)
}
