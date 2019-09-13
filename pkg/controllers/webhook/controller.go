package webhook

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	github2 "github.com/google/go-github/v28/github"
	webhookv1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
	webhookcontrollerv1 "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
	webhookv1controller "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
	"github.com/rancher/gitwatcher/pkg/provider"
	"github.com/rancher/gitwatcher/pkg/provider/github"
	"github.com/rancher/gitwatcher/pkg/provider/polling"
	"github.com/rancher/gitwatcher/pkg/types"
	corev1controller "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/ticker"
	"github.com/sirupsen/logrus"
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
		httpClient:      http.DefaultClient,
		secretCache:     rContext.Core.Core().V1().Secret().Cache(),
	}

	apply := rContext.Apply.WithCacheTypes(
		rContext.Webhook.Gitwatcher().V1().GitWatcher(),
		rContext.Webhook.Gitwatcher().V1().GitCommit())
	wh.providers = append(wh.providers, github.NewGitHub(apply, rContext.Webhook.Gitwatcher().V1().GitCommit(), wh.gitWatcher, secretsLister))
	wh.providers = append(wh.providers, polling.NewPolling(secretsLister, apply))

	rContext.Webhook.Gitwatcher().V1().GitWatcher().OnChange(ctx, "webhook-receiver",
		webhookv1controller.UpdateGitWatcherOnChange(rContext.Webhook.Gitwatcher().V1().GitWatcher().Updater(), wh.onChange))

	rContext.Webhook.Gitwatcher().V1().GitCommit().OnChange(ctx, "gitcommit-github-deployment-status", wh.updateGithubStatus)

	wh.start()
	return nil
}

type webhookHandler struct {
	ctx             context.Context
	gitWatcher      webhookcontrollerv1.GitWatcherController
	gitWatcherCache webhookcontrollerv1.GitWatcherCache
	secretCache     corev1controller.SecretCache
	providers       []provider.Provider
	httpClient      *http.Client
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

func (w *webhookHandler) updateGithubStatus(key string, obj *webhookv1.GitCommit) (*webhookv1.GitCommit, error) {
	if obj == nil || obj.DeletionTimestamp != nil {
		return obj, nil
	}

	if obj.Status.GithubStatus == nil {
		return obj, nil
	}

	gitwatcher, err := w.gitWatcherCache.Get(obj.Namespace, obj.Spec.GitWatcherName)
	if err != nil {
		return obj, err
	}

	secretName := github.DefaultSecretName
	if gitwatcher.Spec.GithubWebhookToken != "" {
		secretName = gitwatcher.Spec.GithubWebhookToken
	}

	secret, err := w.secretCache.Get(obj.Namespace, secretName)
	if err != nil {
		return nil, err
	}

	githubClient := github.NewGithubClient(w.ctx, w.httpClient, string(secret.Data["accessToken"]))

	env := "production"
	if obj.Spec.PR != "" {
		env = "staging"
	}
	autoInactive := true
	if obj.Spec.PR != "" {
		autoInactive = false
	}
	owner, repo, err := github.GetOwnerAndRepo(gitwatcher.Spec.RepositoryURL)
	if err != nil {
		return obj, err
	}
	state := obj.Status.GithubStatus.DeploymentState
	if state == "" {
		state = obj.Status.BuildStatus
	}
	if state == "" {
		return obj, nil
	}
	logrus.Infof("updating deployment %v to state %s", obj.Status.GithubStatus.DeploymentID, state)
	_, resp, err := githubClient.Repositories.CreateDeploymentStatus(context.Background(), owner, repo, obj.Status.GithubStatus.DeploymentID, &github2.DeploymentStatusRequest{
		State:          &state,
		EnvironmentURL: &obj.Status.GithubStatus.EnvironmentURL,
		Environment:    &env,
		AutoInactive:   &autoInactive,
		LogURL:         &obj.Status.GithubStatus.LogURL,
	})
	if err != nil {
		return obj, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		msg, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return obj, err
		}
		return obj, fmt.Errorf("failed to create deployment status, code: %v, error: %v", resp.StatusCode, msg)
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
