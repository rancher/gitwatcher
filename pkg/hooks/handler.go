package hooks

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/gorilla/mux"
	"github.com/rancher/rancher/pkg/ref"
	webhookv1 "github.com/rancher/rio/pkg/apis/webhookinator.rio.cattle.io/v1"
	corev1controller "github.com/rancher/rio/pkg/generated/controllers/core/v1"
	webhookv1controller "github.com/rancher/rio/pkg/generated/controllers/webhookinator.rio.cattle.io/v1"
	"github.com/rancher/webhookinator/pkg/utils"
	"github.com/rancher/webhookinator/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

const (
	githubUrl = "https://api.github.com"
)

type WebhookHandler struct {
	namespace               string
	gitWebHookReceiverCache webhookv1controller.GitWebHookReceiverCache
	gitWebHookExecutions    webhookv1controller.GitWebHookExecutionController
	secretCache             corev1controller.SecretCache
}

func newHandler(rContext *types.Context) *WebhookHandler {
	webhookHandler := &WebhookHandler{
		namespace:               rContext.Namespace,
		gitWebHookReceiverCache: rContext.Webhook.Webhookinator().V1().GitWebHookReceiver().Cache(),
		gitWebHookExecutions:    rContext.Webhook.Webhookinator().V1().GitWebHookExecution(),
		secretCache:             rContext.Core.Core().V1().Secret().Cache(),
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
	receiverID := req.URL.Query().Get(utils.GitWebHookParam)
	ns, name := ref.Parse(receiverID)
	receiver, err := h.gitWebHookReceiverCache.Get(ns, name)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !receiver.Spec.Enabled {
		return http.StatusUnavailableForLegalReasons, errors.New("webhook receiver is disabled")
	}
	credentialID := receiver.Spec.RepositoryCredentialSecretName
	secret, err := h.secretCache.Get(receiver.Namespace, credentialID)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	token := base64.StdEncoding.EncodeToString(secret.Data["accessToken"])
	client, err := newGithubClient(token)
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

func (h *WebhookHandler) validateAndGenerateExecution(webhook scm.Webhook, receiver *webhookv1.GitWebHookReceiver) (int, error) {
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
		Kind:       "GitWebHookReceiver",
		Name:       receiver.Name,
		UID:        receiver.UID,
	})
	_, err := h.gitWebHookExecutions.Create(execution)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func initExecution(receiver *webhookv1.GitWebHookReceiver) *webhookv1.GitWebHookExecution {
	execution := &webhookv1.GitWebHookExecution{}
	execution.GenerateName = receiver.Name + "-"
	execution.Namespace = receiver.Namespace
	execution.Spec.GitWebHookReceiverName = ref.Ref(receiver)
	execution.Labels = receiver.Spec.ExecutionLabels
	execution.Spec.RepositoryURL = receiver.Spec.RepositoryURL
	return execution
}

func newGithubClient(token string) (*scm.Client, error) {
	c, err := github.New(githubUrl)
	if err != nil {
		return nil, err
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	c.Client = tc
	return c, nil
}

func HandleHooks(ctx *types.Context) http.Handler {
	root := mux.NewRouter()
	hooksHandler := newHandler(ctx)
	root.UseEncodedPath()
	root.PathPrefix("/hooks").Handler(hooksHandler)
	return root
}
