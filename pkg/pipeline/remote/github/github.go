package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/rancher/norman/httperror"
	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/model"
	"github.com/rancher/webhookinator/pkg/pipeline/utils"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"github.com/sirupsen/logrus"
	"github.com/tomnomnom/linkheader"
)

const (
	defaultGithubAPI      = "https://api.github.com"
	defaultGithubHost     = "github.com"
	maxPerPage            = "100"
	gheAPI                = "/api/v3"
	hookConfigURL         = "url"
	hookConfigContentType = "content_type"
	hookConfigSecret      = "secret"
	hookConfigInsecureSSL = "insecure_ssl"

	statusPending = "pending"
	statusSuccess = "success"
	statusFailure = "failure"
	descPending   = "This build is pending"
	descSuccess   = "This build is success"
	descFailure   = "This build is failure"
)

type client struct {
	Scheme       string
	Host         string
	ClientID     string
	ClientSecret string
	API          string
}

var defaultClient = &client{
	Scheme: "https://",
	Host:   defaultGithubHost,
	API:    defaultGithubAPI,
}

func New(config *v3.GithubPipelineConfig) (model.Remote, error) {
	if config == nil {
		return defaultClient, nil
	}
	ghClient := &client{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
	}
	if config.Hostname != "" && config.Hostname != defaultGithubHost {
		ghClient.Host = config.Hostname
		if config.TLS {
			ghClient.Scheme = "https://"
		} else {
			ghClient.Scheme = "http://"
		}
		ghClient.API = ghClient.Scheme + ghClient.Host + gheAPI
	} else {
		ghClient.Scheme = "https://"
		ghClient.Host = defaultGithubHost
		ghClient.API = defaultGithubAPI
	}
	return ghClient, nil
}

func (c *client) Type() string {
	return model.GithubType
}

func (c *client) CreateHook(receiver *v1.GitWebHookReceiver, accessToken string) error {
	user, repo, err := getUserRepoFromURL(receiver.Spec.RepositoryURL)
	if err != nil {
		return err
	}

	hookURL := fmt.Sprintf("%s/%s%s", settings.ServerURL.Get(), utils.HooksEndpointPrefix, ref.Ref(receiver))
	events := []string{utils.WebhookEventPush, utils.WebhookEventPullRequest}
	name := "web"
	active := true
	hook := &github.Hook{
		Name:   &name,
		Active: &active,
		Config: map[string]interface{}{
			hookConfigURL:         hookURL,
			hookConfigContentType: "json",
			hookConfigSecret:      receiver.Status.Token,
			hookConfigInsecureSSL: "1",
		},
		Events: events,
	}
	url := fmt.Sprintf("%s/repos/%s/%s/hooks", c.API, user, repo)
	logrus.Debugf("hook to create:%v", hook)
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(hook)

	_, err = doRequestToGithub(http.MethodPost, url, accessToken, b)
	return err
}

func (c *client) DeleteHook(receiver *v1.GitWebHookReceiver, accessToken string) error {
	user, repo, err := getUserRepoFromURL(receiver.Spec.RepositoryURL)
	if err != nil {
		return err
	}

	hook, err := c.getHook(receiver, accessToken)
	if err != nil {
		return err
	}
	if hook != nil {
		url := fmt.Sprintf("%s/repos/%s/%s/hooks/%v", c.API, user, repo, hook.GetID())
		resp, err := doRequestToGithub(http.MethodDelete, url, accessToken, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}
	return nil
}

func (c *client) UpdateStatus(execution *v1.GitWebHookExecution, accessToken string) error {
	user, repo, err := getUserRepoFromURL(execution.Spec.RepositoryURL)
	if err != nil {
		return err
	}
	targetURL := execution.Status.StatusURL
	status, desc := convertStatusDesc(execution)
	context := utils.StatusContext
	commit := execution.Spec.Commit
	githubStatus := &github.RepoStatus{
		State:       &status,
		TargetURL:   &targetURL,
		Description: &desc,
		Context:     &context,
	}
	url := fmt.Sprintf("%s/repos/%s/%s/statuses/%s", c.API, user, repo, commit)
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(githubStatus)

	_, err = doRequestToGithub(http.MethodPost, url, accessToken, b)
	return err
}

func convertStatusDesc(execution *v1.GitWebHookExecution) (string, string) {
	handleCondition := v1.GitWebHookExecutionConditionHandled.GetStatus(execution)
	switch handleCondition {
	case "True":
		return statusSuccess, descSuccess
	case "False":
		return statusFailure, descFailure
	default:
		return statusPending, descPending
	}
}

func (c *client) getHook(receiver *v1.GitWebHookReceiver, accessToken string) (*github.Hook, error) {
	user, repo, err := getUserRepoFromURL(receiver.Spec.RepositoryURL)
	if err != nil {
		return nil, err
	}

	var hooks []github.Hook
	var result *github.Hook
	url := fmt.Sprintf("%s/repos/%s/%s/hooks", c.API, user, repo)

	resp, err := getFromGithub(url, accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &hooks); err != nil {
		return nil, err
	}
	for _, hook := range hooks {
		payloadURL, ok := hook.Config["url"].(string)
		if ok && strings.HasSuffix(payloadURL, fmt.Sprintf("%s%s", utils.HooksEndpointPrefix, ref.Ref(receiver))) {
			result = &hook
		}
	}
	return result, nil
}

func getFromGithub(url string, accessToken string) (*http.Response, error) {
	return doRequestToGithub(http.MethodGet, url, accessToken, nil)
}

func doRequestToGithub(method string, url string, accessToken string, body io.Reader) (*http.Response, error) {
	logrus.Debug("doRequestToGithub", method, url)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	//set to max 100 per page to reduce query time
	if method == http.MethodGet {
		q := req.URL.Query()
		q.Set("per_page", maxPerPage)
		req.URL.RawQuery = q.Encode()
	}
	if accessToken != "" {
		req.Header.Add("Authorization", "token "+accessToken)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36)")
	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	// Check the status code
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		var body bytes.Buffer
		io.Copy(&body, resp.Body)
		return resp, httperror.NewAPIErrorLong(resp.StatusCode, "", body.String())
	}

	return resp, nil
}

func nextGithubPage(response *http.Response) string {
	header := response.Header.Get("link")

	if header != "" {
		links := linkheader.Parse(header)
		for _, link := range links {
			if link.Rel == "next" {
				return link.URL
			}
		}
	}

	return ""
}

func getUserRepoFromURL(repoURL string) (string, string, error) {
	reg := regexp.MustCompile(".*/([^/]*?)/([^/]*?).git")
	match := reg.FindStringSubmatch(repoURL)
	if len(match) != 3 {
		return "", "", fmt.Errorf("error getting user/repo from gitrepoUrl:%v", repoURL)
	}
	return match[1], match[2], nil
}
