package hooks

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	webhookv1controller "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
	"github.com/rancher/gitwatcher/pkg/provider"
	"github.com/rancher/gitwatcher/pkg/provider/github"
	"github.com/rancher/gitwatcher/pkg/types"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/json"
)

type WebhookHandler struct {
	gitWatcherCache webhookv1controller.GitWatcherCache
	gitCommit       webhookv1controller.GitCommitController
	providers       []provider.Provider
}

func newHandler(rContext *types.Context) *WebhookHandler {
	secretCache := rContext.Core.Core().V1().Secret().Cache()
	wh := &WebhookHandler{
		gitWatcherCache: rContext.Webhook.Gitwatcher().V1().GitWatcher().Cache(),
		gitCommit:       rContext.Webhook.Gitwatcher().V1().GitCommit(),
	}
	wh.providers = append(wh.providers, github.NewGitHub(rContext.Apply, wh.gitCommit, rContext.Webhook.Gitwatcher().V1().GitWatcher(), secretCache))
	return wh
}

func (h *WebhookHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	code, err := h.execute(req)
	if err != nil {
		e := map[string]interface{}{
			"type":    "error",
			"code":    code,
			"message": err.Error(),
		}
		logrus.Debugf("executing webhook request got error: %v", err)
		rw.WriteHeader(code)
		responseBody, err := json.Marshal(e)
		if err != nil {
			logrus.Errorf("Failed to unmarshall response, error: %v", err)
		}
		_, err = rw.Write(responseBody)
		if err != nil {
			logrus.Errorf("Failed to write response, error: %v", err)
		}
	}
}

func (h *WebhookHandler) execute(req *http.Request) (int, error) {
	for _, provider := range h.providers {
		code, err := provider.HandleHook(req.Context(), req)
		if err != nil {
			return code, err
		}
		return code, nil
	}
	return http.StatusNotFound, fmt.Errorf("unknown provider")
}

func HandleHooks(ctx *types.Context) http.Handler {
	root := mux.NewRouter()
	hooksHandler := newHandler(ctx)
	logsHander := logsHandler{
		core: ctx.K8s.CoreV1(),
	}
	root.UseEncodedPath()
	root.PathPrefix("/hooks").Handler(hooksHandler)
	root.PathPrefix("/logs").Handler(logsHander)
	return root
}
