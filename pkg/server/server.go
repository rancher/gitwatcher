package server

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rancher/norman"
	"github.com/rancher/norman/types"
	"github.com/rancher/rancher/pkg/pipeline/providers"
	"github.com/rancher/types/config"
	"github.com/rancher/webhookinator/pkg/controllers/execution"
	"github.com/rancher/webhookinator/pkg/controllers/webhook"
	"github.com/rancher/webhookinator/pkg/hooks"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
)

func Config(scaledContext *config.ScaledContext) *norman.Config {
	providers.SetupSourceCodeProviderConfig(scaledContext, scaledContext.Schemas)
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
				if err := webhook.Register(ctx, scaledContext); err != nil {
					return err
				}
				return execution.Register(ctx, scaledContext)
			},
		},
	}
}

func HandleHooks(apiHandler http.Handler, client v1.Interface) http.Handler {
	root := mux.NewRouter()
	hooksHandler := hooks.New(client)
	root.UseEncodedPath()
	root.Handle("/", apiHandler)
	root.PathPrefix("/meta").Handler(apiHandler)
	root.PathPrefix("/v1-webhook").Handler(apiHandler)
	root.PathPrefix("/hooks").Handler(hooksHandler)
	return root
}
