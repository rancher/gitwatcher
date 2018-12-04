package server

import (
	"context"
	"github.com/rancher/norman"
	"github.com/rancher/norman/types"
	"github.com/rancher/types/config"
	"github.com/rancher/webhookinator/pkg/controllers/webhook"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
)

func Config(scaledContext *config.ScaledContext) *norman.Config {
	return &norman.Config{
		Name: "webhookinator",
		Schemas: []*types.Schemas{
			v1.Schemas,
		},

		CRDs: map[*types.APIVersion][]string{
			&v1.APIVersion: {
				v1.GitWebHookReceiverGroupVersionKind.Kind,
				v1.GitWebHookExecutionGroupVersionKind.Kind,
			},
		},

		Clients: []norman.ClientFactory{
			v1.Factory,
		},

		MasterControllers: []norman.ControllerRegister{
			func(ctx context.Context) error {
				return webhook.Register(ctx, scaledContext)
			},
		},
	}
}
