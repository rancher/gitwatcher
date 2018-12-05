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
	"k8s.io/apimachinery/pkg/runtime"
)

func Register(ctx context.Context, scaledContext *config.ScaledContext) error {
	client := v1.From(ctx)
	fl := &webhookExecutionLifecycle{
		webhookExecutions:          client.GitWebHookExecutions(""),
		webhookReceiverLister:      client.GitWebHookReceivers("").Controller().Lister(),
		sourceCodeCredentials:      scaledContext.Project.SourceCodeCredentials(""),
		sourceCodeCredentialLister: scaledContext.Project.SourceCodeCredentials("").Controller().Lister(),
	}

	client.GitWebHookExecutions("").AddHandler(ctx, "webhookexecution-syncer", fl.Sync)
	return nil
}

type webhookExecutionLifecycle struct {
	webhookExecutions          v1.GitWebHookExecutionInterface
	webhookReceiverLister      v1.GitWebHookReceiverLister
	sourceCodeCredentials      v3.SourceCodeCredentialInterface
	sourceCodeCredentialLister v3.SourceCodeCredentialLister
}

func (f *webhookExecutionLifecycle) Sync(key string, obj *v1.GitWebHookExecution) (runtime.Object, error) {
	if obj == nil || obj.DeletionTimestamp != nil {
		return obj, nil
	}
	return obj, f.updateStatus(obj)
}

func (f *webhookExecutionLifecycle) updateStatus(obj *v1.GitWebHookExecution) error {
	appliedStatus := obj.Status.AppliedStatus
	handledStatus := v1.GitWebHookExecutionConditionHandled.GetStatus(obj)
	if handledStatus == appliedStatus {
		return nil
	}
	receiverID := obj.Spec.GitWebHookReceiverName
	ns, name := ref.Parse(receiverID)
	receiver, err := f.webhookReceiverLister.Get(ns, name)
	if err != nil {
		return err
	}
	credentialID := receiver.Spec.RepositoryCredentialSecretName
	ns, name = ref.Parse(credentialID)
	credential, err := f.sourceCodeCredentialLister.Get(ns, name)
	if err != nil {
		return err
	}
	accessToken := credential.Spec.AccessToken
	scpConfig, err := providers.GetSourceCodeProviderConfig(receiver.Spec.Provider, receiver.Namespace)
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
	if err := provider.UpdateStatus(obj, accessToken); err != nil {
		return err
	}
	toUpdate := obj.DeepCopy()
	toUpdate.Status.AppliedStatus = handledStatus
	_, err = f.webhookExecutions.Update(toUpdate)
	return err
}
