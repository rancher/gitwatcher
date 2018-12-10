package utils

import (
	"fmt"
	"net/url"

	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
)

// EnsureAccessToken Checks expiry and do token refresh when needed
//func EnsureAccessToken(credentialInterface v3.SourceCodeCredentialInterface, provider model.Provider, credential *v3.SourceCodeCredential) (string, error) {
//	if credential == nil {
//		return "", nil
//	}
//	refresher, ok := provider.(model.Refresher)
//	if !ok {
//		return credential.Spec.AccessToken, nil
//	}
//
//	t, err := time.Parse(time.RFC3339, credential.Spec.Expiry)
//	if err != nil {
//		return "", err
//	}
//	if t.Before(time.Now().Add(time.Minute)) {
//		torefresh := credential.DeepCopy()
//		ok, err := refresher.Refresh(torefresh)
//		if err != nil {
//			return "", err
//		}
//		if ok {
//			if _, err := credentialInterface.Update(torefresh); err != nil {
//				return "", err
//			}
//		}
//		return torefresh.Spec.AccessToken, nil
//	}
//	return credential.Spec.AccessToken, nil
//}

func GetHookEndpoint(receiver *v1.GitWebHookReceiver) string {
	serverURL := settings.ServerURL.Get()
	//FIXME rancher endpoint that proxy to webhookinator
	serverURL = "http://xxx.ngrok.io"
	return fmt.Sprintf("%s/%s%s", serverURL, HooksEndpointPrefix, ref.Ref(receiver))
}

//TODO test
func GetRepoNameFromURL(repoURL string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", err
	}
	return u.Path, nil
}
