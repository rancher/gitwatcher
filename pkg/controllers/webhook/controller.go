package webhook

import (
	"context"

	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

func Register(ctx context.Context, client v1.Interface) error {
	fl := &webhookReceiverLifecycle{
		webhookReceiverClient: client.GitWebHookReceivers(""),
		webhookReceiverLister: client.GitWebHookReceivers("").Controller().Lister(),
	}

	client.GitWebHookReceivers("").AddHandler(ctx, "webhookReceiver controller", SyncHandler)
	client.GitWebHookReceivers("").AddLifecycle(ctx, "webhookReceiver controller", fl)
	return nil
}

func SyncHandler(key string, obj *v1.GitWebHookReceiver) (runtime.Object, error) {
	// Called anytime something changes, obj will be nil on delete
	logrus.Infof("Sync handler called %s %v", key, obj)
	return obj, nil
}

type webhookReceiverLifecycle struct {
	webhookReceiverClient v1.GitWebHookReceiverInterface
	webhookReceiverLister v1.GitWebHookReceiverLister
}

func (f *webhookReceiverLifecycle) Create(obj *v1.GitWebHookReceiver) (runtime.Object, error) {
	logrus.Infof("Created: %v", obj)
	return obj, nil
}

func (f *webhookReceiverLifecycle) Remove(obj *v1.GitWebHookReceiver) (runtime.Object, error) {
	logrus.Infof("Finalizer: %v", obj)
	return obj, nil
}

func (f *webhookReceiverLifecycle) Updated(obj *v1.GitWebHookReceiver) (runtime.Object, error) {
	logrus.Infof("Updated: %v", obj)
	return obj, nil
}
