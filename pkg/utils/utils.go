package utils

import (
	"time"

	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/webhookinator/pkg/providers/model"
)

// EnsureAccessToken Checks expiry and do token refresh when needed
func EnsureAccessToken(credentialInterface v3.SourceCodeCredentialInterface, provider model.Provider, credential *v3.SourceCodeCredential) (string, error) {
	if credential == nil {
		return "", nil
	}
	refresher, ok := provider.(model.Refresher)
	if !ok {
		return credential.Spec.AccessToken, nil
	}

	t, err := time.Parse(time.RFC3339, credential.Spec.Expiry)
	if err != nil {
		return "", err
	}
	if t.Before(time.Now().Add(time.Minute)) {
		torefresh := credential.DeepCopy()
		ok, err := refresher.Refresh(torefresh)
		if err != nil {
			return "", err
		}
		if ok {
			if _, err := credentialInterface.Update(torefresh); err != nil {
				return "", err
			}
		}
		return torefresh.Spec.AccessToken, nil
	}
	return credential.Spec.AccessToken, nil
}
