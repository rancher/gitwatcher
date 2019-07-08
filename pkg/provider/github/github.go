package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	errors2 "k8s.io/apimachinery/pkg/api/errors"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/google/uuid"
	webhookv1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
	v1 "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
	"github.com/rancher/gitwatcher/pkg/provider/polling"
	"github.com/rancher/gitwatcher/pkg/provider/scmprovider"
	"github.com/rancher/gitwatcher/pkg/utils"
	v12 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/kv"
	"golang.org/x/oauth2"
)

const (
	githubURL           = "https://api.github.com"
	HooksEndpointPrefix = "hooks?gitwebhookId="
	GitWebHookParam     = "gitwebhookId"
	secretName          = "githubtoken"
)

type GitHub struct {
	scmprovider.SCM
	apply apply.Apply
}

func NewGitHub(secrets v12.SecretCache, apply apply.Apply) *GitHub {
	return &GitHub{
		SCM: scmprovider.SCM{
			SecretsCache: secrets,
		},
		apply: apply.WithStrictCaching(),
	}
}

func (w *GitHub) Supports(obj *webhookv1.GitWatcher) bool {
	_, err := w.GetSecret(secretName, obj)
	if errors2.IsNotFound(err) {
		return false
	}

	if strings.EqualFold(obj.Spec.Provider, "github") {
		return true
	}

	if strings.HasPrefix(obj.Spec.RepositoryURL, "https://github.com") {
		return true
	}

	return false
}

func (w *GitHub) Create(ctx context.Context, obj *webhookv1.GitWatcher) (*webhookv1.GitWatcher, error) {
	if obj.Status.HookID != "" {
		return obj, nil
	}

	scmClient, err := w.getClient(obj)
	if err != nil {
		return obj, err
	}
	defer scmClient.Client.CloseIdleConnections()

	obj, err = w.createHook(obj, scmClient)
	if err != nil {
		return obj, err
	}

	return w.getFirstCommit(ctx, obj, scmClient)
}

func (w *GitHub) getFirstCommit(ctx context.Context, obj *webhookv1.GitWatcher, scmClient *scm.Client) (*webhookv1.GitWatcher, error) {
	if obj.Status.FirstCommit != "" || obj.Spec.Branch == "" {
		return obj, nil
	}

	repoName, err := getRepoNameFromURL(obj.Spec.RepositoryURL)
	if err != nil {
		return obj, err
	}

	ref, resp, err := scmClient.Git.FindBranch(ctx, repoName, obj.Spec.Branch)
	if err != nil || resp.Status != http.StatusOK {
		return obj, err
	}

	err = polling.ApplyCommit(obj, ref.Sha, w.apply)
	obj = obj.DeepCopy()
	obj.Status.FirstCommit = ref.Sha
	return obj, err
}

func (w *GitHub) createHook(obj *webhookv1.GitWatcher, scmClient *scm.Client) (*webhookv1.GitWatcher, error) {
	if obj.Status.HookID != "" {
		return obj, nil
	}

	obj = obj.DeepCopy()
	obj.Status.Token = uuid.New().String()

	repoName, err := getRepoNameFromURL(obj.Spec.RepositoryURL)
	if err != nil {
		return obj, err
	}

	in := &scm.HookInput{
		Name:   "rio-gitwatcher",
		Target: getHookEndpoint(obj, obj.Spec.ReceiverURL),
		Secret: obj.Status.Token,
		Events: scm.HookEvents{
			Push:        true,
			Tag:         true,
			PullRequest: obj.Spec.PR,
		},
	}

	hook, _, err := scmClient.Repositories.CreateHook(context.Background(), repoName, in)
	if err != nil {
		return obj, err
	}

	obj.Status.HookID = hook.ID
	return obj, nil
}

func (w *GitHub) getClient(obj *webhookv1.GitWatcher) (*scm.Client, error) {
	secret, err := w.GetSecret(secretName, obj)
	if err != nil {
		return nil, err
	}

	return newGithubClient(string(secret.Data["accessToken"]))
}

func (w *GitHub) HandleHook(gitCommits v1.GitWatcherCache, req *http.Request) (*webhookv1.GitWatcher, scm.Webhook, bool, int, error) {
	receiverID := req.URL.Query().Get(utils.GitWebHookParam)
	if receiverID == "" {
		return nil, nil, false, 0, nil
	}

	ns, name := kv.Split(receiverID, ":")
	receiver, err := gitCommits.Get(ns, name)
	if err != nil {
		return nil, nil, true, http.StatusInternalServerError, err
	}

	if !receiver.Spec.Enabled {
		return nil, nil, true, http.StatusUnavailableForLegalReasons, errors.New("webhook receiver is disabled")
	}

	client, err := w.getClient(receiver)
	if err != nil {
		return nil, nil, true, http.StatusInternalServerError, err
	}
	defer client.Client.CloseIdleConnections()

	f := func(webhook scm.Webhook) (string, error) {
		return receiver.Status.Token, nil
	}
	webhook, err := client.Webhooks.Parse(req, f)
	if err != nil {
		return nil, nil, true, http.StatusBadRequest, err
	}

	return receiver, webhook, true, 0, nil
}

func newGithubClient(token string) (*scm.Client, error) {
	c, err := github.New(githubURL)
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

func getRepoNameFromURL(repoURL string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", err
	}
	repo := strings.TrimPrefix(u.Path, "/")
	repo = strings.TrimSuffix(repo, ".git")
	return repo, nil
}

func getHookEndpoint(receiver *webhookv1.GitWatcher, endpoint string) string {
	if os.Getenv("RIO_WEBHOOK_URL") != "" {
		return hookURL(os.Getenv("RIO_WEBHOOK_URL"), receiver)
	}
	return hookURL(endpoint, receiver)
}

func hookURL(base string, receiver *webhookv1.GitWatcher) string {
	return fmt.Sprintf("%s/%s%s:%s", base, HooksEndpointPrefix, receiver.Namespace, receiver.Name)
}
