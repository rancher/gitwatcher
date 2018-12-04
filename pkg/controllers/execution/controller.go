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

var expectedStatus = map[string]string{
	"True":    "",
	"False":   "",
	"Unknown": "pending",
}

func Register(ctx context.Context, scaledContext *config.ScaledContext) error {
	client := v1.From(ctx)
	fl := &executionLifecycle{
		webhookExecutionLister:     client.GitWebHookExecutions("").Controller().Lister(),
		webhookReceiverClient:      client.GitWebHookReceivers(""),
		webhookReceiverLister:      client.GitWebHookReceivers("").Controller().Lister(),
		sourceCodeCredentials:      scaledContext.Project.SourceCodeCredentials(""),
		sourceCodeCredentialLister: scaledContext.Project.SourceCodeCredentials("").Controller().Lister(),
	}

	client.GitWebHookExecutions("").AddHandler(ctx, "execution-syncer", fl.Sync)
	client.GitWebHookExecutions("").AddLifecycle(ctx, "execution-lifecycle", fl)
	return nil
}

type executionLifecycle struct {
	webhookExecutionLister     v1.GitWebHookExecutionLister
	webhookReceiverClient      v1.GitWebHookReceiverInterface
	webhookReceiverLister      v1.GitWebHookReceiverLister
	sourceCodeCredentialLister v3.SourceCodeCredentialLister
	sourceCodeCredentials      v3.SourceCodeCredentialInterface
}

func (f *executionLifecycle) Sync(key string, obj *v1.GitWebHookExecution) (runtime.Object, error) {
	if obj == nil || obj.DeletionTimestamp != nil {
		return obj, nil
	}
	return obj, nil
}

func (f *executionLifecycle) Create(obj *v1.GitWebHookExecution) (runtime.Object, error) {
	return obj, nil
}

func (f *executionLifecycle) Remove(obj *v1.GitWebHookExecution) (runtime.Object, error) {
	return obj, nil
}

func (f *executionLifecycle) Updated(obj *v1.GitWebHookExecution) (runtime.Object, error) {
	return obj, nil
}

func (f *executionLifecycle) updateStatus(obj *v1.GitWebHookExecution) error {
	//handled := v1.GitWebHookExecutionConditionHandled.GetStatus(obj)
	//appliedStatus := obj.Status.AppliedStatus

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
	return remote.CreateHook(obj, accessToken)
}
