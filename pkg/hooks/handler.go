package hooks

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/drone/go-scm/scm"
	"github.com/gorilla/mux"
	webhookv1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
	webhookv1controller "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
	"github.com/rancher/gitwatcher/pkg/provider"
	"github.com/rancher/gitwatcher/pkg/provider/github"
	"github.com/rancher/gitwatcher/pkg/types"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

type WebhookHandler struct {
	gitWatcherCache webhookv1controller.GitWatcherCache
	gitCommit       webhookv1controller.GitCommitClient
	providers       []provider.Provider
}

func newHandler(rContext *types.Context) *WebhookHandler {
	secretCache := rContext.Core.Core().V1().Secret().Cache()
	wh := &WebhookHandler{
		gitWatcherCache: rContext.Webhook.Gitwatcher().V1().GitWatcher().Cache(),
		gitCommit:       rContext.Webhook.Gitwatcher().V1().GitCommit(),
	}
	wh.providers = append(wh.providers, github.NewGitHub(secretCache, rContext.Apply))
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
		receiver, webhook, ok, code, err := provider.HandleHook(h.gitWatcherCache, req)
		if err != nil {
			return code, err
		}

		if ok {
			return h.validateAndGenerateExecution(webhook, receiver)
		}
	}
	return http.StatusNotFound, fmt.Errorf("unknown provider")
}

func (h *WebhookHandler) validateAndGenerateExecution(webhook scm.Webhook, receiver *webhookv1.GitWatcher) (int, error) {
	execution := initExecution(receiver)
	switch parsed := webhook.(type) {
	case *scm.PushHook:
		if !receiver.Spec.Push {
			return http.StatusUnavailableForLegalReasons, errors.New("push event is deactivated")
		}
		if strings.HasPrefix(parsed.Ref, "refs/heads/") {
			execution.Spec.Branch = strings.TrimPrefix(parsed.Ref, "refs/heads/")
		}
		execution.Spec.Author = parsed.Sender.Login
		execution.Spec.AuthorEmail = parsed.Sender.Email
		execution.Spec.AuthorAvatar = parsed.Sender.Avatar
		execution.Spec.Message = parsed.Commit.Message
		execution.Spec.Commit = parsed.Commit.Sha
		execution.Spec.SourceLink = parsed.Commit.Link
	case *scm.TagHook:
		if !receiver.Spec.Tag {
			return http.StatusUnavailableForLegalReasons, errors.New("tag event is deactivated")
		}
		if parsed.Action != scm.ActionCreate {
			return http.StatusUnavailableForLegalReasons, errors.New("action ommitted")
		}
		execution.Spec.Author = parsed.Sender.Login
		execution.Spec.AuthorEmail = parsed.Sender.Email
		execution.Spec.AuthorAvatar = parsed.Sender.Avatar
		execution.Spec.Tag = parsed.Ref.Name
		execution.Spec.Message = fmt.Sprintf("tag %s is created", parsed.Ref.Name)
		execution.Spec.SourceLink = parsed.Repo.Link
	case *scm.PullRequestHook:
		if !receiver.Spec.PR {
			return http.StatusUnavailableForLegalReasons, errors.New("pull request event is deactivated")
		}
		if parsed.Action != scm.ActionOpen && parsed.Action != scm.ActionSync {
			return http.StatusUnavailableForLegalReasons, errors.New("action ommitted")
		}
		execution.Spec.Author = parsed.Sender.Login
		execution.Spec.AuthorEmail = parsed.Sender.Email
		execution.Spec.AuthorAvatar = parsed.Sender.Avatar
		execution.Spec.PR = strconv.Itoa(parsed.PullRequest.Number)
		execution.Spec.Title = parsed.PullRequest.Title
		execution.Spec.Message = parsed.PullRequest.Body
		execution.Spec.SourceLink = parsed.PullRequest.Link
	}
	execution.OwnerReferences = append(execution.OwnerReferences, metav1.OwnerReference{
		APIVersion: webhookv1.SchemeGroupVersion.String(),
		Kind:       "GitWatcher",
		Name:       receiver.Name,
		UID:        receiver.UID,
	})
	_, err := h.gitCommit.Create(execution)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func initExecution(receiver *webhookv1.GitWatcher) *webhookv1.GitCommit {
	execution := &webhookv1.GitCommit{}
	execution.GenerateName = receiver.Name + "-"
	execution.Namespace = receiver.Namespace
	execution.Spec.GitWebHookReceiverName = receiver.Name
	execution.Labels = receiver.Spec.ExecutionLabels
	execution.Spec.RepositoryURL = receiver.Spec.RepositoryURL
	return execution
}

func HandleHooks(ctx *types.Context) http.Handler {
	root := mux.NewRouter()
	hooksHandler := newHandler(ctx)
	root.UseEncodedPath()
	root.PathPrefix("/hooks").Handler(hooksHandler)
	return root
}
