package scmclient

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/bitbucket"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/driver/stash"
	"github.com/mrjones/oauth"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"golang.org/x/oauth2"
)

const (
	defaultGithubAPI = "https://api.github.com"
	defaultGitlabAPI = "https://gitlab.com"
)

func newDefaultClient() (*scm.Client, error) {
	return github.NewDefault(), nil
}

func newGithubClient(config *v3.GithubPipelineConfig, credential *v3.SourceCodeCredential) (*scm.Client, error) {
	url := ""
	if config.Hostname == "" || config.Hostname == "github.com" {
		url = defaultGithubAPI
	} else if config.TLS {
		url = "https://" + config.Hostname
	} else {
		url = "http://" + config.Hostname
	}
	c, err := github.New(url)
	if err != nil {
		return nil, err
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: credential.Spec.AccessToken},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	c.Client = tc
	return c, nil
}

func newGitlabClient(config *v3.GitlabPipelineConfig, credential *v3.SourceCodeCredential) (*scm.Client, error) {
	url := ""
	if config.Hostname == "" || config.Hostname == "gitlab.com" {
		url = defaultGitlabAPI
	} else if config.TLS {
		url = "https://" + config.Hostname
	} else {
		url = "http://" + config.Hostname
	}
	c, err := gitlab.New(url)
	if err != nil {
		return nil, err
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: credential.Spec.AccessToken},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	c.Client = tc
	return c, nil
}

func newBitbucketCloudClient(config *v3.BitbucketCloudPipelineConfig, credential *v3.SourceCodeCredential) (*scm.Client, error) {
	c := bitbucket.NewDefault()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: credential.Spec.AccessToken},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	c.Client = tc
	return c, nil
}

func newBitbucketServerClient(config *v3.BitbucketServerPipelineConfig, credential *v3.SourceCodeCredential) (*scm.Client, error) {
	url := ""
	if config.Hostname == "" {
		return nil, errors.New("bitbucket server host is not configured")
	} else if config.TLS {
		url = "https://" + config.Hostname
	} else {
		url = "http://" + config.Hostname
	}
	c, err := stash.New(url)
	if err != nil {
		return nil, err
	}
	consumer, err := getOauthConsumer(config.PrivateKey, config.ConsumerKey)
	if err != nil {
		return nil, err
	}
	var token oauth.AccessToken
	token.Token = credential.Spec.AccessToken
	tc, err := consumer.MakeHttpClient(&token)
	if err != nil {
		return nil, err
	}
	c.Client = tc
	return c, nil
}

func getOauthConsumer(privateKey string, consumerKey string) (*oauth.Consumer, error) {
	keyBytes := []byte(privateKey)
	block, _ := pem.Decode(keyBytes)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	oauthConsumer := oauth.NewRSAConsumer(consumerKey, key, oauth.ServiceProvider{})
	return oauthConsumer, nil
}
