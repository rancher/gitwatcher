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
	GitWebHookExecutionGroupVersionKind = schema.GroupVersionKind{
		Version: Version,
		Group:   GroupName,
		Kind:    "GitWebHookExecution",
	}
	GitWebHookExecutionResource = metav1.APIResource{
		Name:         "gitwebhookexecutions",
		SingularName: "gitwebhookexecution",
		Namespaced:   true,

		Kind: GitWebHookExecutionGroupVersionKind.Kind,
	}
)

type GitWebHookExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitWebHookExecution
}

type GitWebHookExecutionHandlerFunc func(key string, obj *GitWebHookExecution) (runtime.Object, error)

type GitWebHookExecutionLister interface {
	List(namespace string, selector labels.Selector) (ret []*GitWebHookExecution, err error)
	Get(namespace, name string) (*GitWebHookExecution, error)
}

type GitWebHookExecutionController interface {
	Generic() controller.GenericController
	Informer() cache.SharedIndexInformer
	Lister() GitWebHookExecutionLister
	AddHandler(ctx context.Context, name string, handler GitWebHookExecutionHandlerFunc)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, handler GitWebHookExecutionHandlerFunc)
	Enqueue(namespace, name string)
	Sync(ctx context.Context) error
	Start(ctx context.Context, threadiness int) error
}

type GitWebHookExecutionInterface interface {
	ObjectClient() *objectclient.ObjectClient
	Create(*GitWebHookExecution) (*GitWebHookExecution, error)
	GetNamespaced(namespace, name string, opts metav1.GetOptions) (*GitWebHookExecution, error)
	Get(name string, opts metav1.GetOptions) (*GitWebHookExecution, error)
	Update(*GitWebHookExecution) (*GitWebHookExecution, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (*GitWebHookExecutionList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Controller() GitWebHookExecutionController
	AddHandler(ctx context.Context, name string, sync GitWebHookExecutionHandlerFunc)
	AddLifecycle(ctx context.Context, name string, lifecycle GitWebHookExecutionLifecycle)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync GitWebHookExecutionHandlerFunc)
	AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle GitWebHookExecutionLifecycle)
}

type gitWebHookExecutionLister struct {
	controller *gitWebHookExecutionController
}

func (l *gitWebHookExecutionLister) List(namespace string, selector labels.Selector) (ret []*GitWebHookExecution, err error) {
	err = cache.ListAllByNamespace(l.controller.Informer().GetIndexer(), namespace, selector, func(obj interface{}) {
		ret = append(ret, obj.(*GitWebHookExecution))
	})
	return
}

func (l *gitWebHookExecutionLister) Get(namespace, name string) (*GitWebHookExecution, error) {
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
			Group:    GitWebHookExecutionGroupVersionKind.Group,
			Resource: "gitWebHookExecution",
		}, key)
	}
	return obj.(*GitWebHookExecution), nil
}

type gitWebHookExecutionController struct {
	controller.GenericController
}

func (c *gitWebHookExecutionController) Generic() controller.GenericController {
	return c.GenericController
}

func (c *gitWebHookExecutionController) Lister() GitWebHookExecutionLister {
	return &gitWebHookExecutionLister{
		controller: c,
	}
}

func (c *gitWebHookExecutionController) AddHandler(ctx context.Context, name string, handler GitWebHookExecutionHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*GitWebHookExecution); ok {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

func (c *gitWebHookExecutionController) AddClusterScopedHandler(ctx context.Context, name, cluster string, handler GitWebHookExecutionHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*GitWebHookExecution); ok && controller.ObjectInCluster(cluster, obj) {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

type gitWebHookExecutionFactory struct {
}

func (c gitWebHookExecutionFactory) Object() runtime.Object {
	return &GitWebHookExecution{}
}

func (c gitWebHookExecutionFactory) List() runtime.Object {
	return &GitWebHookExecutionList{}
}

func (s *gitWebHookExecutionClient) Controller() GitWebHookExecutionController {
	s.client.Lock()
	defer s.client.Unlock()

	c, ok := s.client.gitWebHookExecutionControllers[s.ns]
	if ok {
		return c
	}

	genericController := controller.NewGenericController(GitWebHookExecutionGroupVersionKind.Kind+"Controller",
		s.objectClient)

	c = &gitWebHookExecutionController{
		GenericController: genericController,
	}

	s.client.gitWebHookExecutionControllers[s.ns] = c
	s.client.starters = append(s.client.starters, c)

	return c
}

type gitWebHookExecutionClient struct {
	client       *Client
	ns           string
	objectClient *objectclient.ObjectClient
	controller   GitWebHookExecutionController
}

func (s *gitWebHookExecutionClient) ObjectClient() *objectclient.ObjectClient {
	return s.objectClient
}

func (s *gitWebHookExecutionClient) Create(o *GitWebHookExecution) (*GitWebHookExecution, error) {
	obj, err := s.objectClient.Create(o)
	return obj.(*GitWebHookExecution), err
}

func (s *gitWebHookExecutionClient) Get(name string, opts metav1.GetOptions) (*GitWebHookExecution, error) {
	obj, err := s.objectClient.Get(name, opts)
	return obj.(*GitWebHookExecution), err
}

func (s *gitWebHookExecutionClient) GetNamespaced(namespace, name string, opts metav1.GetOptions) (*GitWebHookExecution, error) {
	obj, err := s.objectClient.GetNamespaced(namespace, name, opts)
	return obj.(*GitWebHookExecution), err
}

func (s *gitWebHookExecutionClient) Update(o *GitWebHookExecution) (*GitWebHookExecution, error) {
	obj, err := s.objectClient.Update(o.Name, o)
	return obj.(*GitWebHookExecution), err
}

func (s *gitWebHookExecutionClient) Delete(name string, options *metav1.DeleteOptions) error {
	return s.objectClient.Delete(name, options)
}

func (s *gitWebHookExecutionClient) DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error {
	return s.objectClient.DeleteNamespaced(namespace, name, options)
}

func (s *gitWebHookExecutionClient) List(opts metav1.ListOptions) (*GitWebHookExecutionList, error) {
	obj, err := s.objectClient.List(opts)
	return obj.(*GitWebHookExecutionList), err
}

func (s *gitWebHookExecutionClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return s.objectClient.Watch(opts)
}

// Patch applies the patch and returns the patched deployment.
func (s *gitWebHookExecutionClient) Patch(o *GitWebHookExecution, data []byte, subresources ...string) (*GitWebHookExecution, error) {
	obj, err := s.objectClient.Patch(o.Name, o, data, subresources...)
	return obj.(*GitWebHookExecution), err
}

func (s *gitWebHookExecutionClient) DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return s.objectClient.DeleteCollection(deleteOpts, listOpts)
}

func (s *gitWebHookExecutionClient) AddHandler(ctx context.Context, name string, sync GitWebHookExecutionHandlerFunc) {
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *gitWebHookExecutionClient) AddLifecycle(ctx context.Context, name string, lifecycle GitWebHookExecutionLifecycle) {
	sync := NewGitWebHookExecutionLifecycleAdapter(name, false, s, lifecycle)
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *gitWebHookExecutionClient) AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync GitWebHookExecutionHandlerFunc) {
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}

func (s *gitWebHookExecutionClient) AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle GitWebHookExecutionLifecycle) {
	sync := NewGitWebHookExecutionLifecycleAdapter(name+"_"+clusterName, true, s, lifecycle)
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}
