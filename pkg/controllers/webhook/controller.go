package webhook

import (
	"context"

	"github.com/rancher/rancher/pkg/pipeline/providers"
	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/types/config"
	remoteproviders "github.com/rancher/webhookinator/pkg/providers"
	"github.com/rancher/webhookinator/pkg/utils"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"github.com/satori/go.uuid"
	"k8s.io/apimachinery/pkg/runtime"
)

func Register(ctx context.Context, scaledContext *config.ScaledContext) error {
	client := v1.From(ctx)
	fl := &webhookReceiverLifecycle{
		webhookReceivers:           client.GitWebHookReceivers(""),
		webhookReceiverLister:      client.GitWebHookReceivers("").Controller().Lister(),
		sourceCodeCredentials:      scaledContext.Project.SourceCodeCredentials(""),
		sourceCodeCredentialLister: scaledContext.Project.SourceCodeCredentials("").Controller().Lister(),
	}

	client.GitWebHookReceivers("").AddLifecycle(ctx, "webhookreceiver-lifecycle", fl)
	return nil
}

type webhookReceiverLifecycle struct {
	webhookReceivers           v1.GitWebHookReceiverInterface
	webhookReceiverLister      v1.GitWebHookReceiverLister
	sourceCodeCredentialLister v3.SourceCodeCredentialLister
	sourceCodeCredentials      v3.SourceCodeCredentialInterface
}

func (f *webhookReceiverLifecycle) Create(obj *v1.GitWebHookReceiver) (runtime.Object, error) {
	return obj, nil
}

func (f *webhookReceiverLifecycle) Remove(obj *v1.GitWebHookReceiver) (runtime.Object, error) {
	return obj, f.deleteHook(obj)
}

func (f *webhookReceiverLifecycle) Updated(obj *v1.GitWebHookReceiver) (runtime.Object, error) {
	if obj.Status.Token == "" {
		//random token for webhook validation
		obj.Status.Token = uuid.NewV4().String()
	}
	newObj, err := v1.GitWebHookReceiverConditionRegistered.Once(obj, func() (runtime.Object, error) {
		return obj, f.createHook(obj)
	})
	if err != nil {
		return obj, err
	}
	return newObj, nil
}

func (f *webhookReceiverLifecycle) createHook(obj *v1.GitWebHookReceiver) error {
	credentialID := obj.Spec.RepositoryCredentialSecretName
	ns, name := ref.Parse(credentialID)
	credential, err := f.sourceCodeCredentialLister.Get(ns, name)
	if err != nil {
		return err
	}
	accessToken := credential.Spec.AccessToken
	scpConfig, err := providers.GetSourceCodeProviderConfig(obj.Spec.Provider, obj.Namespace)
	if err != nil {
		return err
	}
	provider, err := remoteproviders.New(scpConfig)
	if err != nil {
		return err
	}
	accessToken, err = utils.EnsureAccessToken(f.sourceCodeCredentials, provider, credential)
	if err != nil {
		return err
	}
	return provider.CreateHook(obj, accessToken)
}

func (f *webhookReceiverLifecycle) deleteHook(obj *v1.GitWebHookReceiver) error {
	credentialID := obj.Spec.RepositoryCredentialSecretName
	ns, name := ref.Parse(credentialID)
	credential, err := f.sourceCodeCredentialLister.Get(ns, name)
	if err != nil {
		return err
	}
	accessToken := credential.Spec.AccessToken
	scpConfig, err := providers.GetSourceCodeProviderConfig(obj.Spec.Provider, obj.Namespace)
	if err != nil {
		return err
	}
	provider, err := remoteproviders.New(scpConfig)
	if err != nil {
		return err
	}
	accessToken, err = utils.EnsureAccessToken(f.sourceCodeCredentials, provider, credential)
	if err != nil {
		return err
	}
	return provider.DeleteHook(obj, accessToken)
}
