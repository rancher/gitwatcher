package webhook

import (
	"context"

	"github.com/drone/go-scm/scm"
	"github.com/rancher/rancher/pkg/pipeline/providers"
	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/rancher/webhookinator/pkg/scmclient"
	"github.com/rancher/webhookinator/pkg/utils"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	credential, err := f.sourceCodeCredentials.GetNamespaced(ns, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	scpConfig, err := providers.GetSourceCodeProviderConfig(obj.Spec.Provider, obj.Namespace)
	if err != nil {
		return err
	}
	credential, err = utils.EnsureAccessToken(f.sourceCodeCredentials, scpConfig, credential)
	if err != nil {
		return err
	}
	client, err := scmclient.NewClient(scpConfig, credential)
	if err != nil {
		return err
	}
	repoName, err := utils.GetRepoNameFromURL(obj.Spec.RepositoryURL)
	if err != nil {
		return err
	}
	in := &scm.HookInput{
		Name:   "rancher-webhookinator",
		Target: utils.GetHookEndpoint(obj),
		Secret: obj.Status.Token,
		Events: scm.HookEvents{
			Push:        true,
			Tag:         true,
			PullRequest: true,
		},
	}
	hook, _, err := client.Repositories.CreateHook(context.Background(), repoName, in)
	if err != nil {
		return err
	}
	obj.Status.HookID = hook.ID
	return nil
}

func (f *webhookReceiverLifecycle) deleteHook(obj *v1.GitWebHookReceiver) error {
	credentialID := obj.Spec.RepositoryCredentialSecretName
	ns, name := ref.Parse(credentialID)
	credential, err := f.sourceCodeCredentials.GetNamespaced(ns, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	scpConfig, err := providers.GetSourceCodeProviderConfig(obj.Spec.Provider, obj.Namespace)
	if err != nil {
		return err
	}
	credential, err = utils.EnsureAccessToken(f.sourceCodeCredentials, scpConfig, credential)
	if err != nil {
		return err
	}
	client, err := scmclient.NewClient(scpConfig, credential)
	if err != nil {
		return err
	}
	repoName, err := utils.GetRepoNameFromURL(obj.Spec.RepositoryURL)
	if err != nil {
		return err
	}
	if obj.Status.HookID == "" {
		return nil
	}
	_, err = client.Repositories.DeleteHook(context.Background(), repoName, obj.Status.HookID)
	return err
}
