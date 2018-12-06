package bitbucketcloud

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

	"github.com/pkg/errors"
	"github.com/rancher/norman/httperror"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/webhookinator/pkg/providers/model"
	"github.com/rancher/webhookinator/pkg/utils"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"golang.org/x/oauth2"
)

const (
	apiEndpoint      = "https://api.bitbucket.org/2.0"
	authURL          = "https://bitbucket.org/site/oauth2/authorize"
	tokenURL         = "https://bitbucket.org/site/oauth2/access_token"
	maxPerPage       = "100"
	statusInProgress = "INPROGRESS"
	statusSuccessful = "SUCCESSFUL"
	statusFailed     = "FAILED"
	descInProgress   = "This build is in progress"
	descSuccessful   = "This build is successful"
	descFailed       = "This build is failed"
)

type client struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func New(config *v3.BitbucketCloudPipelineConfig) (model.Provider, error) {
	if config == nil {
		return nil, errors.New("empty bitbucket cloud config")
	}
	glClient := &client{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
	}
	return glClient, nil
}

func (c *client) Type() string {
	return model.BitbucketCloudType
}

func (c *client) CreateHook(receiver *v1.GitWebHookReceiver, accessToken string) error {
	user, repo, err := getUserRepoFromURL(receiver.Spec.RepositoryURL)
	if err != nil {
		return err
	}
	hook := Hook{
		Description:          "Webhook created by Rancher Pipeline",
		URL:                  utils.GetHookEndpoint(receiver),
		Active:               true,
		SkipCertVerification: true,
		Events: []string{
			"repo:push",
			"pullrequest:updated",
			"pullrequest:created",
		},
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/hooks", apiEndpoint, user, repo)
	b, err := json.Marshal(hook)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(b)

	_, err = doRequestToBitbucket(http.MethodPost, url, accessToken, nil, reader)
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
		url := fmt.Sprintf("%s/repositories/%s/%s/hooks/%v", apiEndpoint, user, repo, hook.UUID)
		_, err := doRequestToBitbucket(http.MethodDelete, url, accessToken, nil, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *client) UpdateStatus(execution *v1.GitWebHookExecution, accessToken string) error {
	user, repo, err := getUserRepoFromURL(execution.Spec.RepositoryURL)
	if err != nil {
		return err
	}
	commit := execution.Spec.Commit
	state, desc := convertStatusDesc(execution)
	status := Status{
		Key:         utils.StatusContext,
		URL:         execution.Status.StatusURL,
		State:       state,
		Description: desc,
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/commit/%s/statuses/build", apiEndpoint, user, repo, commit)
	b, err := json.Marshal(status)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(b)

	_, err = doRequestToBitbucket(http.MethodPost, url, accessToken, nil, reader)
	return err
}

func convertStatusDesc(execution *v1.GitWebHookExecution) (string, string) {
	handleCondition := v1.GitWebHookExecutionConditionHandled.GetStatus(execution)
	switch handleCondition {
	case "True":
		return statusSuccessful, descSuccessful
	case "False":
		return statusFailed, descFailed
	default:
		return statusInProgress, descInProgress
	}
}

func (c *client) getHook(receiver *v1.GitWebHookReceiver, accessToken string) (*Hook, error) {
	user, repo, err := getUserRepoFromURL(receiver.Spec.RepositoryURL)
	if err != nil {
		return nil, err
	}

	var hooks PaginatedHooks
	var result *Hook
	url := fmt.Sprintf("%s/repositories/%s/%s/hooks", apiEndpoint, user, repo)

	b, err := getFromBitbucket(url, accessToken)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &hooks); err != nil {
		return nil, err
	}
	for _, hook := range hooks.Values {
		if strings.HasSuffix(hook.URL, utils.GetHookEndpointSuffix(receiver)) {
			result = &hook
			break
		}
	}
	return result, nil
}

func (c *client) Refresh(cred *v3.SourceCodeCredential) (bool, error) {
	if cred == nil {
		return false, errors.New("cannot refresh empty credentials")
	}
	config := &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
	source := config.TokenSource(
		oauth2.NoContext, &oauth2.Token{RefreshToken: cred.Spec.RefreshToken})

	token, err := source.Token()
	if err != nil || len(token.AccessToken) == 0 {
		return false, err
	}

	cred.Spec.AccessToken = token.AccessToken
	cred.Spec.RefreshToken = token.RefreshToken
	cred.Spec.Expiry = token.Expiry.Format(time.RFC3339)

	return true, nil

}

func getFromBitbucket(url string, accessToken string) ([]byte, error) {
	return doRequestToBitbucket(http.MethodGet, url, accessToken, nil, nil)
}

func doRequestToBitbucket(method string, url string, accessToken string, header map[string]string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	q := req.URL.Query()
	//set to max 100 per page to reduce query time
	if method == http.MethodGet {
		q.Set("pagelen", maxPerPage)
	}
	if accessToken != "" {
		q.Set("access_token", accessToken)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-control", "no-cache")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Check the status code
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		var body bytes.Buffer
		io.Copy(&body, resp.Body)
		return nil, httperror.NewAPIErrorLong(resp.StatusCode, "", body.String())
	}
	r, err := ioutil.ReadAll(resp.Body)
	return r, err
}

func getUserRepoFromURL(repoURL string) (string, string, error) {
	reg := regexp.MustCompile(".*/([^/]*?)/([^/]*?).git")
	match := reg.FindStringSubmatch(repoURL)
	if len(match) != 3 {
		return "", "", fmt.Errorf("error getting user/repo from gitrepoUrl:%v", repoURL)
	}
	return match[1], match[2], nil
}
