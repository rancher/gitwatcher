package v1

import (
	"context"

	"github.com/rancher/norman/controller"
	"github.com/rancher/norman/objectclient"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

var (
	GitWebHookReceiverGroupVersionKind = schema.GroupVersionKind{
		Version: Version,
		Group:   GroupName,
		Kind:    "GitWebHookReceiver",
	}
	GitWebHookReceiverResource = metav1.APIResource{
		Name:         "gitwebhookreceivers",
		SingularName: "gitwebhookreceiver",
		Namespaced:   true,

		Kind: GitWebHookReceiverGroupVersionKind.Kind,
	}
)

type GitWebHookReceiverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitWebHookReceiver
}

type GitWebHookReceiverHandlerFunc func(key string, obj *GitWebHookReceiver) (runtime.Object, error)

type GitWebHookReceiverLister interface {
	List(namespace string, selector labels.Selector) (ret []*GitWebHookReceiver, err error)
	Get(namespace, name string) (*GitWebHookReceiver, error)
}

type GitWebHookReceiverController interface {
	Generic() controller.GenericController
	Informer() cache.SharedIndexInformer
	Lister() GitWebHookReceiverLister
	AddHandler(ctx context.Context, name string, handler GitWebHookReceiverHandlerFunc)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, handler GitWebHookReceiverHandlerFunc)
	Enqueue(namespace, name string)
	Sync(ctx context.Context) error
	Start(ctx context.Context, threadiness int) error
}

type GitWebHookReceiverInterface interface {
	ObjectClient() *objectclient.ObjectClient
	Create(*GitWebHookReceiver) (*GitWebHookReceiver, error)
	GetNamespaced(namespace, name string, opts metav1.GetOptions) (*GitWebHookReceiver, error)
	Get(name string, opts metav1.GetOptions) (*GitWebHookReceiver, error)
	Update(*GitWebHookReceiver) (*GitWebHookReceiver, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (*GitWebHookReceiverList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Controller() GitWebHookReceiverController
	AddHandler(ctx context.Context, name string, sync GitWebHookReceiverHandlerFunc)
	AddLifecycle(ctx context.Context, name string, lifecycle GitWebHookReceiverLifecycle)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync GitWebHookReceiverHandlerFunc)
	AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle GitWebHookReceiverLifecycle)
}

type gitWebHookReceiverLister struct {
	controller *gitWebHookReceiverController
}

func (l *gitWebHookReceiverLister) List(namespace string, selector labels.Selector) (ret []*GitWebHookReceiver, err error) {
	err = cache.ListAllByNamespace(l.controller.Informer().GetIndexer(), namespace, selector, func(obj interface{}) {
		ret = append(ret, obj.(*GitWebHookReceiver))
	})
	return
}

func (l *gitWebHookReceiverLister) Get(namespace, name string) (*GitWebHookReceiver, error) {
	var key string
	if namespace != "" {
		key = namespace + "/" + name
	} else {
		key = name
	}
	obj, exists, err := l.controller.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    GitWebHookReceiverGroupVersionKind.Group,
			Resource: "gitWebHookReceiver",
		}, key)
	}
	return obj.(*GitWebHookReceiver), nil
}

type gitWebHookReceiverController struct {
	controller.GenericController
}

func (c *gitWebHookReceiverController) Generic() controller.GenericController {
	return c.GenericController
}

func (c *gitWebHookReceiverController) Lister() GitWebHookReceiverLister {
	return &gitWebHookReceiverLister{
		controller: c,
	}
}

func (c *gitWebHookReceiverController) AddHandler(ctx context.Context, name string, handler GitWebHookReceiverHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*GitWebHookReceiver); ok {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

func (c *gitWebHookReceiverController) AddClusterScopedHandler(ctx context.Context, name, cluster string, handler GitWebHookReceiverHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*GitWebHookReceiver); ok && controller.ObjectInCluster(cluster, obj) {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

type gitWebHookReceiverFactory struct {
}

func (c gitWebHookReceiverFactory) Object() runtime.Object {
	return &GitWebHookReceiver{}
}

func (c gitWebHookReceiverFactory) List() runtime.Object {
	return &GitWebHookReceiverList{}
}

func (s *gitWebHookReceiverClient) Controller() GitWebHookReceiverController {
	s.client.Lock()
	defer s.client.Unlock()

	c, ok := s.client.gitWebHookReceiverControllers[s.ns]
	if ok {
		return c
	}

	genericController := controller.NewGenericController(GitWebHookReceiverGroupVersionKind.Kind+"Controller",
		s.objectClient)

	c = &gitWebHookReceiverController{
		GenericController: genericController,
	}

	s.client.gitWebHookReceiverControllers[s.ns] = c
	s.client.starters = append(s.client.starters, c)

	return c
}

type gitWebHookReceiverClient struct {
	client       *Client
	ns           string
	objectClient *objectclient.ObjectClient
	controller   GitWebHookReceiverController
}

func (s *gitWebHookReceiverClient) ObjectClient() *objectclient.ObjectClient {
	return s.objectClient
}

func (s *gitWebHookReceiverClient) Create(o *GitWebHookReceiver) (*GitWebHookReceiver, error) {
	obj, err := s.objectClient.Create(o)
	return obj.(*GitWebHookReceiver), err
}

func (s *gitWebHookReceiverClient) Get(name string, opts metav1.GetOptions) (*GitWebHookReceiver, error) {
	obj, err := s.objectClient.Get(name, opts)
	return obj.(*GitWebHookReceiver), err
}

func (s *gitWebHookReceiverClient) GetNamespaced(namespace, name string, opts metav1.GetOptions) (*GitWebHookReceiver, error) {
	obj, err := s.objectClient.GetNamespaced(namespace, name, opts)
	return obj.(*GitWebHookReceiver), err
}

func (s *gitWebHookReceiverClient) Update(o *GitWebHookReceiver) (*GitWebHookReceiver, error) {
	obj, err := s.objectClient.Update(o.Name, o)
	return obj.(*GitWebHookReceiver), err
}

func (s *gitWebHookReceiverClient) Delete(name string, options *metav1.DeleteOptions) error {
	return s.objectClient.Delete(name, options)
}

func (s *gitWebHookReceiverClient) DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error {
	return s.objectClient.DeleteNamespaced(namespace, name, options)
}

func (s *gitWebHookReceiverClient) List(opts metav1.ListOptions) (*GitWebHookReceiverList, error) {
	obj, err := s.objectClient.List(opts)
	return obj.(*GitWebHookReceiverList), err
}

func (s *gitWebHookReceiverClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return s.objectClient.Watch(opts)
}

// Patch applies the patch and returns the patched deployment.
func (s *gitWebHookReceiverClient) Patch(o *GitWebHookReceiver, data []byte, subresources ...string) (*GitWebHookReceiver, error) {
	obj, err := s.objectClient.Patch(o.Name, o, data, subresources...)
	return obj.(*GitWebHookReceiver), err
}

func (s *gitWebHookReceiverClient) DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return s.objectClient.DeleteCollection(deleteOpts, listOpts)
}

func (s *gitWebHookReceiverClient) AddHandler(ctx context.Context, name string, sync GitWebHookReceiverHandlerFunc) {
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *gitWebHookReceiverClient) AddLifecycle(ctx context.Context, name string, lifecycle GitWebHookReceiverLifecycle) {
	sync := NewGitWebHookReceiverLifecycleAdapter(name, false, s, lifecycle)
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *gitWebHookReceiverClient) AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync GitWebHookReceiverHandlerFunc) {
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}

func (s *gitWebHookReceiverClient) AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle GitWebHookReceiverLifecycle) {
	sync := NewGitWebHookReceiverLifecycleAdapter(name+"_"+clusterName, true, s, lifecycle)
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}
