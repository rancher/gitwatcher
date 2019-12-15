package github

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v28/github"
	"github.com/google/uuid"
	webhookv1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
	v1 "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
	"github.com/rancher/gitwatcher/pkg/git"
	"github.com/rancher/gitwatcher/pkg/provider/polling"
	"github.com/rancher/gitwatcher/pkg/utils"
	corev1controller "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/kv"
	"golang.org/x/oauth2"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	githubURL           = "https://api.github.com"
	HooksEndpointPrefix = "hooks?gitwebhookId="
	GitWebHookParam     = "gitwebhookId"
	DefaultSecretName   = "githubtoken"
)

const (
	statusOpened   = "opened"
	statusReopened = "reopened"
	statusClosed   = "closed"
	statusMerged   = "merged"
	statusSynced   = "synchronize"
)

type GitHub struct {
	gitWatchers v1.GitWatcherController
	gitCommits  v1.GitCommitController
	secretCache corev1controller.SecretCache
	httpClient  *http.Client
	apply       apply.Apply
}

func NewGitHub(apply apply.Apply, gitCommits v1.GitCommitController, gitWatchers v1.GitWatcherController, secretCache corev1controller.SecretCache) *GitHub {
	return &GitHub{
		secretCache: secretCache,
		gitCommits:  gitCommits,
		gitWatchers: gitWatchers,
		apply:       apply.WithStrictCaching(),
		httpClient:  http.DefaultClient,
	}
}

func (w *GitHub) Supports(obj *webhookv1.GitWatcher) bool {
	secretName := DefaultSecretName
	if obj.Spec.GithubWebhookToken != "" {
		secretName = obj.Spec.GithubWebhookToken
	}
	_, err := w.secretCache.Get(obj.Namespace, secretName)
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

	githubClient, err := w.getClient(ctx, obj)
	if err != nil {
		return obj, err
	}

	obj, err = w.createHook(ctx, obj, githubClient)
	if err != nil {
		return obj, err
	}

	return w.getFirstCommit(ctx, obj, githubClient)
}

func (w *GitHub) getFirstCommit(ctx context.Context, obj *webhookv1.GitWatcher, client *github.Client) (*webhookv1.GitWatcher, error) {
	if obj.Status.FirstCommit != "" || obj.Spec.Branch == "" {
		return obj, nil
	}

	owner, repo, err := GetOwnerAndRepo(obj.Spec.RepositoryURL)
	if err != nil {
		return obj, err
	}

	ref, resp, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+obj.Spec.Branch)
	if err != nil {
		return obj, fmt.Errorf("failed to get ref for %s/%s, error: %v", owner, repo, err)
	}
	defer resp.Body.Close()

	if resp != nil && resp.StatusCode != http.StatusOK {
		msg, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return obj, fmt.Errorf("failed to get ref, failed to read api response")
		}
		return obj, fmt.Errorf("failed to get ref for %s/%s, error: %v", owner, repo, msg)
	}

	if ref.GetObject() == nil || ref.GetObject().SHA == nil {
		return obj, nil
	}

	err = polling.ApplyCommit(obj, *ref.GetObject().SHA, w.apply)
	obj = obj.DeepCopy()
	obj.Status.FirstCommit = *ref.GetObject().SHA
	return obj, err
}

func (w *GitHub) createHook(ctx context.Context, obj *webhookv1.GitWatcher, client *github.Client) (*webhookv1.GitWatcher, error) {
	if obj.Status.HookID != "" {
		return obj, nil
	}

	obj = obj.DeepCopy()
	obj.Status.Token = uuid.New().String()

	owner, repo, err := GetOwnerAndRepo(obj.Spec.RepositoryURL)
	if err != nil {
		return obj, err
	}

	events := getEvents(obj)
	hook, resp, err := client.Repositories.CreateHook(ctx, owner, repo, &github.Hook{
		Events: events,
		Config: map[string]interface{}{
			"url":    getHookEndpoint(obj, obj.Spec.ReceiverURL),
			"secret": obj.Status.Token,
		},
	})
	if err != nil {
		return obj, fmt.Errorf("failed to create hook for %s/%s, error: %v", owner, repo, err)
	}
	defer resp.Body.Close()

	if resp != nil && resp.StatusCode != http.StatusCreated {
		msg, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return obj, fmt.Errorf("failed to create hook, failed to read api response")
		}
		return obj, fmt.Errorf("failed to create hook for %s/%s, error: %v", owner, repo, msg)
	}

	if hook != nil && hook.ID != nil {
		obj.Status.HookID = strconv.Itoa(int(*hook.ID))
	}

	return obj, nil
}

func getEvents(obj *webhookv1.GitWatcher) []string {
	var events []string
	if obj.Spec.Push {
		events = append(events, "push")
	}

	if obj.Spec.PR {
		events = append(events, "pull_request")
	}

	if obj.Spec.Tag {
		events = append(events, "create")
	}
	return events
}

func (w *GitHub) getClient(ctx context.Context, obj *webhookv1.GitWatcher) (*github.Client, error) {
	secretName := DefaultSecretName
	if obj.Spec.GithubWebhookToken != "" {
		secretName = obj.Spec.GithubWebhookToken
	}

	secret, err := w.secretCache.Get(obj.Namespace, secretName)
	if err != nil {
		return nil, err
	}

	return NewGithubClient(ctx, w.httpClient, string(secret.Data["accessToken"])), nil
}

func (w *GitHub) HandleHook(ctx context.Context, req *http.Request) (int, error) {
	receiverID := req.URL.Query().Get(utils.GitWebHookParam)
	if receiverID == "" {
		return 0, nil
	}

	ns, name := kv.Split(receiverID, ":")
	gitwatcher, err := w.gitWatchers.Get(ns, name, metav1.GetOptions{})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if !gitwatcher.Spec.Enabled {
		return http.StatusUnprocessableEntity, errors.New("webhook receiver is disabled")
	}

	payload, err := github.ValidatePayload(req, []byte(gitwatcher.Status.Token))
	if err != nil {
		return http.StatusInternalServerError, err
	}
	event, err := github.ParseWebHook(github.WebHookType(req), payload)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	client, err := w.getClient(ctx, gitwatcher)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return w.handleEvent(ctx, client, event, gitwatcher)
}

func (w *GitHub) handleEvent(ctx context.Context, client *github.Client, event interface{}, receiver *webhookv1.GitWatcher) (int, error) {
	execution := initExecution(receiver)
	switch event.(type) {
	case *github.CreateEvent:
		if receiver.Spec.Tag == false {
			return http.StatusUnprocessableEntity, fmt.Errorf("tag watching is not currently turned on")
		}
		parsed := event.(*github.CreateEvent)
		if parsed.Ref == nil {
			return http.StatusUnprocessableEntity, errors.New("create event has empty tag ref")
		}
		if parsed.GetRefType() != "tag" {
			return http.StatusUnprocessableEntity, errors.New("create event only supports tag type")
		}
		execution.Spec.Tag = *parsed.Ref
		err := git.TagMatch(receiver.Spec.TagIncludeRegexp, receiver.Spec.TagExcludeRegexp, execution.Spec.Tag)
		if err != nil {
			return http.StatusUnprocessableEntity, err
		}
		if parsed.Sender != nil {
			execution.Spec.Author = safeString(parsed.Sender.Login)
			execution.Spec.AuthorEmail = safeString(parsed.Sender.Email)
			execution.Spec.AuthorAvatar = safeString(parsed.Sender.AvatarURL)
		}

	case *github.PushEvent:
		parsed := event.(*github.PushEvent)
		if parsed.Ref != nil {
			if strings.HasPrefix(*parsed.Ref, "refs/heads/") {
				execution.Spec.Branch = strings.TrimPrefix(*parsed.Ref, "refs/heads/")
			} else {
				return http.StatusUnprocessableEntity, fmt.Errorf("push event only handles commits") // tag should be handled via create event
			}
		}
		if parsed.Sender != nil {
			execution.Spec.Author = safeString(parsed.Sender.Login)
			execution.Spec.AuthorEmail = safeString(parsed.Sender.Email)
			execution.Spec.AuthorAvatar = safeString(parsed.Sender.AvatarURL)
		}

		if parsed.GetHeadCommit() != nil {
			execution.Spec.Message = safeString(parsed.GetHeadCommit().Message)
			execution.Spec.Commit = safeString(parsed.GetHeadCommit().ID)
			execution.Spec.SourceLink = safeString(parsed.GetHeadCommit().URL)
			if execution.Spec.Branch == receiver.Spec.Branch {
				if err := w.createDeploymentForProduction(ctx, client, receiver, execution, safeString(parsed.GetHeadCommit().ID)); err != nil {
					return http.StatusInternalServerError, err
				}
			}
		}
	case *github.PullRequestEvent:
		if !receiver.Spec.PR {
			return http.StatusUnprocessableEntity, fmt.Errorf("pull request is not enabled")
		}
		parsed := event.(*github.PullRequestEvent)
		if parsed.Action != nil && (*parsed.Action != statusOpened && *parsed.Action != statusReopened && *parsed.Action != statusClosed && *parsed.Action != statusMerged && *parsed.Action != statusSynced) {
			return http.StatusUnprocessableEntity, fmt.Errorf("action %s ommitted", *parsed.Action)
		}
		execution.Spec.Action = *parsed.Action
		if parsed.Sender != nil {
			execution.Spec.Author = safeString(parsed.Sender.Login)
			execution.Spec.AuthorEmail = safeString(parsed.Sender.Email)
			execution.Spec.AuthorAvatar = safeString(parsed.Sender.AvatarURL)
		}
		if parsed.Number != nil {
			execution.Spec.PR = strconv.Itoa(*parsed.Number)
		}

		if parsed.PullRequest != nil {
			execution.Spec.Title = safeString(parsed.PullRequest.Title)
			execution.Spec.Message = safeString(parsed.PullRequest.Body)
			execution.Spec.SourceLink = safeString(parsed.PullRequest.URL)
			execution.Spec.Merged = safeBool(parsed.PullRequest.Merged)
			if parsed.PullRequest.Head != nil {
				execution.Spec.Commit = safeString(parsed.PullRequest.Head.SHA)
			}
		}

		if *parsed.Action == statusClosed {
			execution.Spec.Closed = true
		}

		if parsed.Repo != nil {
			execution.Spec.RepositoryURL = safeString(parsed.Repo.HTMLURL)
		}

		if err := w.createDeploymentForPullRequest(ctx, client, receiver, execution, parsed); err != nil {
			return http.StatusInternalServerError, err
		}
	}
	execution.OwnerReferences = append(execution.OwnerReferences, metav1.OwnerReference{
		APIVersion: webhookv1.SchemeGroupVersion.String(),
		Kind:       "GitWatcher",
		Name:       receiver.Name,
		UID:        receiver.UID,
	})
	_, err := w.gitCommits.Create(execution)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (w *GitHub) createDeploymentForProduction(ctx context.Context, client *github.Client, gitWatcher *webhookv1.GitWatcher, gitCommit *webhookv1.GitCommit, commit string) error {
	if !gitWatcher.Spec.GithubDeployment {
		return nil
	}

	owner, repo, err := GetOwnerAndRepo(gitWatcher.Spec.RepositoryURL)
	if err != nil {
		return err
	}

	ref := commit
	environment := "production"
	req := &github.DeploymentRequest{
		Ref:         &ref,
		Environment: &environment,
	}

	deploy, err := w.createDeployment(ctx, client, owner, repo, req)
	if err != nil {
		return err
	}
	if gitCommit.Status.GithubStatus == nil {
		gitCommit.Status.GithubStatus = &webhookv1.GithubStatus{}
	}
	gitCommit.Status.GithubStatus.DeploymentID = *deploy.ID
	return err
}

func (w *GitHub) createDeploymentForPullRequest(ctx context.Context, client *github.Client, gitWatcher *webhookv1.GitWatcher, gitCommit *webhookv1.GitCommit, event *github.PullRequestEvent) error {
	if !gitWatcher.Spec.GithubDeployment {
		return nil
	}

	if *event.Action != statusOpened && *event.Action != statusSynced {
		return nil
	}

	owner, repo, err := GetOwnerAndRepo(gitWatcher.Spec.RepositoryURL)
	if err != nil {
		return err
	}

	if event.PullRequest == nil || event.PullRequest.ID == nil {
		return fmt.Errorf("failed to find pull request data")
	}

	ref := fmt.Sprintf("pull/%v/head", *event.PullRequest.Number)
	req := &github.DeploymentRequest{
		Ref:         &ref,
		Environment: &[]string{"staging"}[0],
	}
	deploy, err := w.createDeployment(ctx, client, owner, repo, req)
	if err != nil {
		return err
	}
	if gitCommit.Status.GithubStatus == nil {
		gitCommit.Status.GithubStatus = &webhookv1.GithubStatus{}
	}
	gitCommit.Status.GithubStatus.DeploymentID = *deploy.ID
	return err
}

func (w *GitHub) createDeployment(ctx context.Context, client *github.Client, owner, repo string, req *github.DeploymentRequest) (*github.Deployment, error) {
	deploy, resp, err := client.Repositories.CreateDeployment(ctx, owner, repo, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment, err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		msg, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to create deployment, code: %v, error: %v", resp.StatusCode, msg)
	}
	return deploy, nil
}

func initExecution(receiver *webhookv1.GitWatcher) *webhookv1.GitCommit {
	execution := &webhookv1.GitCommit{}
	execution.GenerateName = receiver.Name + "-"
	execution.Namespace = receiver.Namespace
	execution.Spec.GitWatcherName = receiver.Name
	execution.Labels = receiver.Spec.ExecutionLabels
	execution.Spec.RepositoryURL = receiver.Spec.RepositoryURL
	return execution
}

func NewGithubClient(ctx context.Context, httpClient *http.Client, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	subCtx := context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	tc := oauth2.NewClient(subCtx, ts)

	client := github.NewClient(tc)
	return client
}

func GetOwnerAndRepo(repoURL string) (string, string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", err
	}
	repo := strings.TrimPrefix(u.Path, "/")
	repo = strings.TrimSuffix(repo, ".git")
	owner, repo := kv.Split(repo, "/")
	return owner, repo, nil
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

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
