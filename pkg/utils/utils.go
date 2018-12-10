package utils

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"golang.org/x/oauth2"
)

const (
	bitbucketCloudTokenURL = "https://bitbucket.org/site/oauth2/access_token"
)

// EnsureAccessToken Checks expiry and do token refresh when needed
func EnsureAccessToken(credentialInterface v3.SourceCodeCredentialInterface, config interface{}, credential *v3.SourceCodeCredential) (*v3.SourceCodeCredential, error) {
	if credential == nil {
		return nil, nil
	}
	bitbucketCloudConfig, ok := config.(*v3.BitbucketCloudPipelineConfig)
	if !ok {
		return credential, nil
	}
	t, err := time.Parse(time.RFC3339, credential.Spec.Expiry)
	if err != nil {
		return nil, err
	}
	if t.Before(time.Now().Add(time.Minute)) {
		config := &oauth2.Config{
			ClientID:     bitbucketCloudConfig.ClientID,
			ClientSecret: bitbucketCloudConfig.ClientSecret,
			Endpoint: oauth2.Endpoint{
				TokenURL: bitbucketCloudTokenURL,
			},
		}
		source := config.TokenSource(
			oauth2.NoContext, &oauth2.Token{RefreshToken: credential.Spec.RefreshToken})

		token, err := source.Token()
		if err != nil || len(token.AccessToken) == 0 {
			return nil, err
		}
		toupdate := credential.DeepCopy()
		toupdate.Spec.AccessToken = token.AccessToken
		toupdate.Spec.RefreshToken = token.RefreshToken
		toupdate.Spec.Expiry = token.Expiry.Format(time.RFC3339)
		if _, err := credentialInterface.Update(toupdate); err != nil {
			return nil, err
		}
		return toupdate, nil
	}
	return credential, nil
}

func GetHookEndpoint(receiver *v1.GitWebHookReceiver) string {
	serverURL := settings.ServerURL.Get()
	//FIXME rancher endpoint that proxy to webhookinator
	serverURL = "http://xx.ngrok.io"
	return fmt.Sprintf("%s/%s%s", serverURL, HooksEndpointPrefix, ref.Ref(receiver))
}

func GetRepoNameFromURL(repoURL string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", err
	}
	repo := strings.TrimPrefix(u.Path, "/")
	repo = strings.TrimSuffix(repo, ".git")
	return repo, nil
}
