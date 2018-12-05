package model

import (
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
)

type Remote interface {
	Type() string

	CreateHook(receiver *v1.GitWebHookReceiver, accessToken string) error
	DeleteHook(receiver *v1.GitWebHookReceiver, accessToken string) error
	UpdateStatus(execution *v1.GitWebHookExecution, accessToken string) error
}

type Refresher interface {
	Refresh(cred *v3.SourceCodeCredential) (bool, error)
}
