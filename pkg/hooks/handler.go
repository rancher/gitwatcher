package hooks

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/drone/go-scm/scm"
	"github.com/rancher/rancher/pkg/pipeline/providers"
	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/webhookinator/pkg/scmclient"
	"github.com/rancher/webhookinator/pkg/utils"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/json"
)

type WebhookHandler struct {
	GitWebHookReceiverLister v1.GitWebHookReceiverLister
	GitWebHookExecutions     v1.GitWebHookExecutionInterface
}

func New(management v1.Interface) *WebhookHandler {
	webhookHandler := &WebhookHandler{
		GitWebHookReceiverLister: management.GitWebHookReceivers("").Controller().Lister(),
		GitWebHookExecutions:     management.GitWebHookExecutions(""),
	}
	return webhookHandler
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
		responseBody, _ := json.Marshal(e)
		rw.Write(responseBody)
	}

}

func (h *WebhookHandler) execute(req *http.Request) (int, error) {
	receiverID := req.URL.Query().Get(utils.GitWebHookParam)
	ns, name := ref.Parse(receiverID)
	receiver, err := h.GitWebHookReceiverLister.Get(ns, name)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !receiver.Spec.Enabled {
		return http.StatusUnavailableForLegalReasons, errors.New("webhook receiver is disabled")
	}
	credentialID := receiver.Spec.RepositoryCredentialSecretName
	ns, name = ref.Parse(credentialID)
	scpConfig, err := providers.GetSourceCodeProviderConfig(receiver.Spec.Provider, receiver.Namespace)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	client, err := scmclient.NewClient(scpConfig)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	f := func(webhook scm.Webhook) (string, error) {
		return receiver.Status.Token, nil
	}
	webhook, err := client.Webhooks.Parse(req, f)
	if err != nil {
		return http.StatusBadRequest, err
	}
	return h.validateAndGenerateExecution(webhook, receiver)
}

func (h *WebhookHandler) validateAndGenerateExecution(webhook scm.Webhook, receiver *v1.GitWebHookReceiver) (int, error) {
	execution := initExecution(receiver)
	switch parsed := webhook.(type) {
	case *scm.PushHook:
		if !receiver.Spec.Push {
			return http.StatusUnavailableForLegalReasons, errors.New("push event is deactivated")
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
	_, err := h.GitWebHookExecutions.Create(execution)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func initExecution(receiver *v1.GitWebHookReceiver) *v1.GitWebHookExecution {
	execution := &v1.GitWebHookExecution{}
	execution.GenerateName = receiver.Name + "-"
	execution.Namespace = receiver.Namespace
	execution.Spec.GitWebHookReceiverName = ref.Ref(receiver)
	execution.Labels = receiver.Spec.ExecutionLabels
	execution.Spec.RepositoryURL = receiver.Spec.RepositoryURL
	return execution
}
