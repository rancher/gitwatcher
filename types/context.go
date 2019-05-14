package types

import (
	"context"

	core "github.com/rancher/rio/pkg/generated/controllers/core"
	webhook "github.com/rancher/rio/pkg/generated/controllers/webhookinator.rio.cattle.io"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/start"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type contextKey struct{}

type Context struct {
	Namespace string

	Webhook *webhook.Factory
	Core    *core.Factory
	K8s     kubernetes.Interface

	Apply apply.Apply
}

func Store(ctx context.Context, c *Context) context.Context {
	return context.WithValue(ctx, contextKey{}, c)
}

func From(ctx context.Context) *Context {
	return ctx.Value(contextKey{}).(*Context)
}

func NewContext(namespace string, config *rest.Config) *Context {
	context := &Context{
		Namespace: namespace,
		Core:      core.NewFactoryFromConfigOrDie(config),
		Webhook:   webhook.NewFactoryFromConfigOrDie(config),
		K8s:       kubernetes.NewForConfigOrDie(config),
	}

	context.Apply = apply.New(context.K8s.Discovery(), apply.NewClientFactory(config))
	return context
}

func (c *Context) Start(ctx context.Context) error {
	return start.All(ctx, 5,
		c.Webhook,
		c.Core,
	)
}

func BuildContext(ctx context.Context, namespace string, config *rest.Config) (context.Context, *Context) {
	c := NewContext(namespace, config)
	return context.WithValue(ctx, contextKey{}, c), c
}
