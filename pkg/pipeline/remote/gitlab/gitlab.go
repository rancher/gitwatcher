package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/pkg/errors"
	"github.com/rancher/norman/httperror"
	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/model"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"github.com/tomnomnom/linkheader"
	"github.com/xanzy/go-gitlab"
)

const (
	defaultGitlabAPI     = "https://gitlab.com/api/v4"
	defaultGitlabHost    = "gitlab.com"
	maxPerPage           = "100"
	gitlabAPI            = "%s%s/api/v4"
	gitlabLoginName      = "oauth2"
	accessLevelReporter  = 20
	accessLevelDeveloper = 30
	accessLevelMaster    = 40
)

type client struct {
	Scheme       string
	Host         string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	API          string
}

func New(config *v3.GitlabPipelineConfig) (model.Remote, error) {
	if config == nil {
		return nil, errors.New("empty gitlab config")
	}
	glClient := &client{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
	}
	if config.Hostname != "" && config.Hostname != defaultGitlabHost {
		glClient.Host = config.Hostname
		if config.TLS {
			glClient.Scheme = "https://"
		} else {
			glClient.Scheme = "http://"
		}
		glClient.API = fmt.Sprintf(gitlabAPI, glClient.Scheme, glClient.Host)
	} else {
		glClient.Scheme = "https://"
		glClient.Host = defaultGitlabHost
		glClient.API = defaultGitlabAPI
	}
	return glClient, nil
}

func (c *client) Type() string {
	return model.GitlabType
}

func (c *client) CreateHook(receiver *v1.GitWebHookReceiver, accessToken string) error {
	user, repo, err := getUserRepoFromURL(receiver.Spec.RepositoryURL)
	if err != nil {
		return err
	}
	project := url.QueryEscape(user + "/" + repo)
	hookURL := fmt.Sprintf("%s/hooks?pipelineId=%s", settings.ServerURL.Get(), ref.Ref(receiver))
	opt := &gitlab.AddProjectHookOptions{
		PushEvents:            gitlab.Bool(true),
		MergeRequestsEvents:   gitlab.Bool(true),
		TagPushEvents:         gitlab.Bool(true),
		URL:                   gitlab.String(hookURL),
		EnableSSLVerification: gitlab.Bool(false),
		Token:                 gitlab.String(receiver.Status.Token),
	}
	url := fmt.Sprintf("%s/projects/%s/hooks", c.API, project)
	_, err = doRequestToGitlab(http.MethodPost, url, accessToken, opt)
	return err
}

func (c *client) DeleteHook(receiver *v1.GitWebHookReceiver, accessToken string) error {
	user, repo, err := getUserRepoFromURL(receiver.Spec.RepositoryURL)
	if err != nil {
		return err
	}
	project := url.QueryEscape(user + "/" + repo)

	hook, err := c.getHook(receiver, accessToken)
	if err != nil {
		return err
	}
	if hook != nil {
		url := fmt.Sprintf("%s/projects/%s/hooks/%v", c.API, project, hook.ID)
		resp, err := doRequestToGitlab(http.MethodDelete, url, accessToken, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}
	return nil
}

func (c *client) UpdateStatus(execution *v1.GitWebHookExecution, status string, accessToken string) error {
	//TODO
	return nil
}

func (c *client) getHook(receiver *v1.GitWebHookReceiver, accessToken string) (*gitlab.ProjectHook, error) {
	user, repo, err := getUserRepoFromURL(receiver.Spec.RepositoryURL)
	if err != nil {
		return nil, err
	}
	project := url.QueryEscape(user + "/" + repo)

	var hooks []gitlab.ProjectHook
	var result *gitlab.ProjectHook
	url := fmt.Sprintf(c.API+"/projects/%s/hooks", project)
	resp, err := getFromGitlab(accessToken, url)
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
		if strings.HasSuffix(hook.URL, fmt.Sprintf("hooks?pipelineId=%s", ref.Ref(receiver))) {
			result = &hook
		}
	}
	return result, nil
}

func getFromGitlab(gitlabAccessToken string, url string) (*http.Response, error) {
	return doRequestToGitlab(http.MethodGet, url, gitlabAccessToken, nil)
}

func doRequestToGitlab(method string, url string, gitlabAccessToken string, opt interface{}) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
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
	if opt != nil {
		q := req.URL.Query()
		optq, err := query.Values(opt)
		if err != nil {
			return nil, err
		}
		for k, v := range optq {
			q[k] = v
		}
		req.URL.RawQuery = q.Encode()
	}
	if gitlabAccessToken != "" {
		req.Header.Add("Authorization", "Bearer "+gitlabAccessToken)
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

func paginateGitlab(gitlabAccessToken string, url string) ([]*http.Response, error) {
	var responses []*http.Response

	response, err := getFromGitlab(gitlabAccessToken, url)
	if err != nil {
		return responses, err
	}
	responses = append(responses, response)
	nextURL := nextGitlabPage(response)
	for nextURL != "" {
		response, err = getFromGitlab(gitlabAccessToken, nextURL)
		if err != nil {
			return responses, err
		}
		responses = append(responses, response)
		nextURL = nextGitlabPage(response)
	}

	return responses, nil
}

func nextGitlabPage(response *http.Response) string {
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

func convertAccount(gitlabAccount *gitlab.User) *v3.SourceCodeCredential {

	if gitlabAccount == nil {
		return nil
	}
	account := &v3.SourceCodeCredential{}
	account.Spec.SourceCodeType = model.GitlabType

	account.Spec.AvatarURL = gitlabAccount.AvatarURL
	account.Spec.HTMLURL = gitlabAccount.WebsiteURL
	account.Spec.LoginName = gitlabAccount.Username
	account.Spec.GitLoginName = gitlabLoginName
	account.Spec.DisplayName = gitlabAccount.Name

	return account

}

func (c *client) getGitlabRepos(gitlabAccessToken string) ([]v3.SourceCodeRepository, error) {
	url := c.API + "/projects?membership=true"
	var repos []gitlab.Project
	responses, err := paginateGitlab(gitlabAccessToken, url)
	if err != nil {
		return nil, err
	}
	for _, response := range responses {
		defer response.Body.Close()
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		var reposObj []gitlab.Project
		if err := json.Unmarshal(b, &reposObj); err != nil {
			return nil, err
		}
		repos = append(repos, reposObj...)
	}

	return convertRepos(repos), nil
}

func convertRepos(repos []gitlab.Project) []v3.SourceCodeRepository {
	result := []v3.SourceCodeRepository{}
	for _, repo := range repos {
		r := v3.SourceCodeRepository{}

		r.Spec.URL = repo.HTTPURLToRepo
		//r.Spec.Language = No language info in gitlab API
		r.Spec.DefaultBranch = repo.DefaultBranch

		accessLevel := getAccessLevel(repo)
		if accessLevel >= accessLevelReporter {
			// 20 for 'Reporter' level
			r.Spec.Permissions.Pull = true
		}
		if accessLevel >= accessLevelDeveloper {
			// 30 for 'Developer' level
			r.Spec.Permissions.Push = true
		}
		if accessLevel >= accessLevelMaster {
			// 40 for 'Master' level and 50 for 'Owner' level
			r.Spec.Permissions.Admin = true
		}
		result = append(result, r)
	}
	return result
}

func getAccessLevel(repo gitlab.Project) int {
	accessLevel := 0
	if repo.Permissions == nil {
		return accessLevel
	}
	if repo.Permissions.ProjectAccess != nil && int(repo.Permissions.ProjectAccess.AccessLevel) > accessLevel {
		accessLevel = int(repo.Permissions.ProjectAccess.AccessLevel)
	}
	if repo.Permissions.GroupAccess != nil && int(repo.Permissions.GroupAccess.AccessLevel) > accessLevel {
		accessLevel = int(repo.Permissions.GroupAccess.AccessLevel)
	}
	return accessLevel
}

func getUserRepoFromURL(repoURL string) (string, string, error) {
	reg := regexp.MustCompile(".*/([^/]*?)/([^/]*?).git")
	match := reg.FindStringSubmatch(repoURL)
	if len(match) != 3 {
		return "", "", fmt.Errorf("error getting user/repo from gitrepoUrl:%v", repoURL)
	}
	return match[1], match[2], nil
}
