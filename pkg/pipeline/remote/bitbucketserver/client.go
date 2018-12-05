package bitbucketserver

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/mrjones/oauth"
	"github.com/pkg/errors"
	"github.com/rancher/norman/httperror"
	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/model"
	"github.com/rancher/webhookinator/pkg/pipeline/utils"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
)

const (
	maxPerPage        = "100"
	requestTokenURL   = "%s/plugins/servlet/oauth/request-token"
	authorizeTokenURL = "%s/plugins/servlet/oauth/authorize"
	accessTokenURL    = "%s/plugins/servlet/oauth/access-token"
	statusInProgress  = "INPROGRESS"
	statusSuccessful  = "SUCCESSFUL"
	statusFailed      = "FAILED"
	descInProgress    = "This build is in progress"
	descSuccessful    = "This build is successful"
	descFailed        = "This build is failed"
)

type client struct {
	BaseURL     string
	ConsumerKey string
	PrivateKey  string
	RedirectURL string
}

func New(config *v3.BitbucketServerPipelineConfig) (model.Remote, error) {
	if config == nil {
		return nil, errors.New("empty bitbucket server config")
	}
	bsClient := &client{
		ConsumerKey: config.ConsumerKey,
		PrivateKey:  config.PrivateKey,
		RedirectURL: config.RedirectURL,
	}
	if config.TLS {
		bsClient.BaseURL = "https://" + config.Hostname
	} else {
		bsClient.BaseURL = "http://" + config.Hostname
	}
	return bsClient, nil
}

func (c *client) Type() string {
	return model.BitbucketServerType
}

func (c *client) getOauthConsumer() (*oauth.Consumer, error) {
	keyBytes := []byte(c.PrivateKey)
	block, _ := pem.Decode(keyBytes)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	bitbucketOauthConsumer := oauth.NewRSAConsumer(
		c.ConsumerKey,
		privateKey,
		oauth.ServiceProvider{
			RequestTokenUrl:   fmt.Sprintf(requestTokenURL, c.BaseURL),
			AuthorizeTokenUrl: fmt.Sprintf(authorizeTokenURL, c.BaseURL),
			AccessTokenUrl:    fmt.Sprintf(accessTokenURL, c.BaseURL),
			HttpMethod:        http.MethodPost,
		})
	return bitbucketOauthConsumer, nil
}

func (c *client) CreateHook(receiver *v1.GitWebHookReceiver, accessToken string) error {
	user, repo, err := getUserRepoFromURL(receiver.Spec.RepositoryURL)
	if err != nil {
		return err
	}
	hookURL := fmt.Sprintf("%s/%s%s", settings.ServerURL.Get(), utils.HooksEndpointPrefix, ref.Ref(receiver))
	hook := Hook{
		Name:   "pipeline webhook",
		URL:    hookURL,
		Active: true,
		Configuration: HookConfiguration{
			Secret: receiver.Status.Token,
		},
		Events: []string{
			"repo:refs_changed",
			"pr:opened",
			"pr:modified",
		},
	}

	url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/webhooks", c.BaseURL, user, repo)
	b, err := json.Marshal(hook)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(b)
	_, err = c.doRequestToBitbucket(http.MethodPost, url, accessToken, nil, reader)
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
		url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/webhooks/%d", c.BaseURL, user, repo, hook.ID)
		_, err := c.doRequestToBitbucket(http.MethodDelete, url, accessToken, nil, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *client) UpdateStatus(execution *v1.GitWebHookExecution, accessToken string) error {
	commit := execution.Spec.Commit
	state, desc := convertStatusDesc(execution)
	status := Status{
		URL:         execution.Status.StatusURL,
		Key:         utils.StatusContext,
		State:       state,
		Description: desc,
	}

	url := fmt.Sprintf("%s/rest/build-status/1.0/commits/%s", c.BaseURL, commit)
	b, err := json.Marshal(status)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(b)
	_, err = c.doRequestToBitbucket(http.MethodPost, url, accessToken, nil, reader)
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
	url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/webhooks", c.BaseURL, user, repo)

	b, err := c.getFromBitbucket(url, accessToken)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &hooks); err != nil {
		return nil, err
	}
	for _, hook := range hooks.Values {
		if strings.HasSuffix(hook.URL, fmt.Sprintf("%s%s", utils.HooksEndpointPrefix, ref.Ref(receiver))) {
			result = &hook
			break
		}
	}
	return result, nil
}

func (c *client) getFromBitbucket(url string, accessToken string) ([]byte, error) {
	return c.doRequestToBitbucket(http.MethodGet, url, accessToken, nil, nil)
}

func (c *client) doRequestToBitbucket(method string, url string, accessToken string, header map[string]string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	consumer, err := c.getOauthConsumer()
	if err != nil {
		return nil, err
	}
	var token oauth.AccessToken
	token.Token = accessToken
	client, err := consumer.MakeHttpClient(&token)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	if method == http.MethodGet {
		q.Set("limit", maxPerPage)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
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
