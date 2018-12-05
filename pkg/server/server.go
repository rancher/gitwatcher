package server

import (
	"context"
	"github.com/rancher/rancher/server/responsewriter"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rancher/norman"
	"github.com/rancher/norman/types"
	"github.com/rancher/types/config"
	"github.com/rancher/webhookinator/pkg/controllers/webhook"
	"github.com/rancher/webhookinator/pkg/hooks"
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

func HandleHooks(handler http.Handler, client v1.Interface) http.Handler {
	root := mux.NewRouter()
	hooksHandler := hooks.New(client)
	chain := responsewriter.NewMiddlewareChain(responsewriter.Gzip, responsewriter.NoCache, responsewriter.ContentType)
	root.Handle("/", chain.Handler(handler))
	root.UseEncodedPath()
	root.Handle("/", handler)
	root.PathPrefix("/hooks").Handler(hooksHandler)
	return root
}
