package webhook

import (
	"context"

	"github.com/rancher/rancher/pkg/pipeline/providers"
	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/rancher/webhookinator/pkg/pipeline/remote"
	"github.com/rancher/webhookinator/pkg/pipeline/utils"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	statusPending = "pending"
	statusSuccess = ""
	statusFailed  = ""
)

var handledToStatus = map[string]string{
	"True":    statusSuccess,
	"False":   statusFailed,
	"Unknown": statusPending,
}

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
	handled := v1.GitWebHookExecutionConditionHandled.GetStatus(obj)
	toApplyStatus := handledToStatus[handled]
	if toApplyStatus == appliedStatus {
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
	remote, err := remote.New(scpConfig)
	if err != nil {
		return err
	}
	accessToken, err = utils.EnsureAccessToken(f.sourceCodeCredentials, remote, credential)
	if err != nil {
		return err
	}
	if err := remote.UpdateStatus(obj, toApplyStatus, accessToken); err != nil {
		return err
	}
	toUpdate := obj.DeepCopy()
	toUpdate.Status.AppliedStatus = toApplyStatus
	_, err = f.webhookExecutions.Update(toUpdate)
	return err
}
