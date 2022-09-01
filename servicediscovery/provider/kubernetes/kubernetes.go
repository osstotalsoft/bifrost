package kubernetes

import (
	"context"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/osstotalsoft/bifrost/servicediscovery"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//KubeServiceProvider is a service discovery provider implementation, using Kubernetes
type KubeServiceProvider struct {
	onAddServiceHandlers    []servicediscovery.ServiceFunc
	onRemoveServiceHandlers []servicediscovery.ServiceFunc
	onUpdateServiceHandlers []func(old servicediscovery.Service, new servicediscovery.Service)
	stop                    chan struct{}
	clientset               *kubernetes.Clientset
	overrideServiceAddress  string
	logger                  log.Logger
	filterFunc              func(name string, namespace string) bool
}

const resourceLabelName = "api-gateway/resource"
const audienceLabelName = "api-gateway/oidc.audience"
const securedLabelName = "api-gateway/secured"

//NewKubernetesServiceDiscoveryProvider creates a new kube provider
func NewKubernetesServiceDiscoveryProvider(inCluster bool, overrideServiceAddress string,
	filterServiceNamespaceByPrefix string, loggerFactory log.Factory) *KubeServiceProvider {

	logger := loggerFactory(nil)
	logger = logger.With(zap.String("component", "kubernetes_service_provider"))

	var config *rest.Config
	var err error

	if inCluster {
		config, err = rest.InClusterConfig()
	} else {
		config, err = outOfClusterConfig()
	}

	if err != nil {
		logger.Panic("KubernetesProvider: cannot connect to discovery provider", zap.Error(err))
	}

	if inCluster && overrideServiceAddress != "" {
		logger.Panic("KubernetesProvider: You cannot override service address while in cluster mode", zap.Error(err))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Panic("KubernetesProvider: cannot connect to discovery provider", zap.Error(err))
	}

	p := &KubeServiceProvider{
		onAddServiceHandlers:    []servicediscovery.ServiceFunc{},
		onRemoveServiceHandlers: []servicediscovery.ServiceFunc{},
		onUpdateServiceHandlers: []func(old servicediscovery.Service, new servicediscovery.Service){},
		clientset:               clientset,
		stop:                    make(chan struct{}),
		overrideServiceAddress:  overrideServiceAddress,
		logger:                  logger,
	}

	if filterServiceNamespaceByPrefix != "" {
		p.filterFunc = func(name string, namespace string) bool {
			return strings.HasPrefix(namespace, filterServiceNamespaceByPrefix)
		}
	}

	return p
}

func outOfClusterConfig() (*rest.Config, error) {
	kubeconfig := filepath.Join(os.Getenv("USERPROFILE"), ".kube", "config")

	// use the current context in kubeconfig
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func (provider *KubeServiceProvider) allowService(name string, namespace string) bool {
	if provider.filterFunc != nil {
		return provider.filterFunc(name, namespace)
	}
	return true
}

//Start starts the discovery process
func Start(provider *KubeServiceProvider) *KubeServiceProvider {
	watchlist := newServicesListWatch(provider.clientset.CoreV1().RESTClient())
	_, controller := cache.NewInformer(watchlist, &corev1.Service{}, time.Second*0, cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc(provider),
		DeleteFunc: deleteFunc(provider),
		UpdateFunc: updateFunc(provider),
	})

	go controller.Run(provider.stop)
	return provider
}

func updateFunc(provider *KubeServiceProvider) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		oldSrv := oldObj.(*corev1.Service)
		newSrv := newObj.(*corev1.Service)

		if !provider.allowService(newSrv.Name, newSrv.Namespace) {
			return
		}
		provider.logger.Info("KubernetesProvider: service updated", zap.Any("old_service", oldSrv), zap.Any("new_service", newSrv))
		callUpdateSubscribers(provider.onUpdateServiceHandlers,
			mapToService(oldSrv, provider.overrideServiceAddress),
			mapToService(newSrv, provider.overrideServiceAddress))
	}
}

func deleteFunc(provider *KubeServiceProvider) func(obj interface{}) {
	return func(obj interface{}) {
		srv := obj.(*corev1.Service)
		if !provider.allowService(srv.Name, srv.Namespace) {
			return
		}
		provider.logger.Info("KubernetesProvider: service deleted", zap.Any("service", srv))
		callSubscribers(provider.onRemoveServiceHandlers, mapToService(srv, provider.overrideServiceAddress))
	}
}

func addFunc(provider *KubeServiceProvider) func(obj interface{}) {
	return func(obj interface{}) {
		srv := obj.(*corev1.Service)
		if !provider.allowService(srv.Name, srv.Namespace) {
			return
		}
		provider.logger.Info("KubernetesProvider: service added", zap.Any("service", srv))
		callSubscribers(provider.onAddServiceHandlers, mapToService(srv, provider.overrideServiceAddress))
	}
}

func mapToService(srv *corev1.Service, overrideServiceAddress string) servicediscovery.Service {
	secured, _ := strconv.ParseBool(srv.Labels[securedLabelName])

	address := "http://" + srv.Name + "." + srv.Namespace
	if overrideServiceAddress != "" {
		address = overrideServiceAddress
	}
	return servicediscovery.Service{
		Address:      address,
		Version:      srv.ResourceVersion,
		UID:          string(srv.UID),
		Name:         srv.Name,
		Resource:     srv.Labels[resourceLabelName],
		OidcAudience: srv.Labels[audienceLabelName],
		Secured:      secured,
		Namespace:    srv.Namespace,
	}
}

// to apply modification to ListOptions with a field selector, a label selector, or any other desired options.
func newServicesListWatch(c cache.Getter) *cache.ListWatch {
	resource := "services"
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = resourceLabelName
		return c.Get().
			//Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Do(context.TODO()).
			Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.LabelSelector = resourceLabelName
		return c.Get().
			//Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Watch(context.TODO())
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func callSubscribers(handlers []servicediscovery.ServiceFunc, service servicediscovery.Service) {
	for _, fn := range handlers {
		fn(service)
	}
}

func callUpdateSubscribers(handlers []func(old servicediscovery.Service, new servicediscovery.Service), old servicediscovery.Service, new servicediscovery.Service) {
	for _, fn := range handlers {
		fn(old, new)
	}
}

//Stop stops the discovery process
func Stop(provider *KubeServiceProvider) *KubeServiceProvider {
	close(provider.stop)
	return provider
}

//SubscribeOnAddService registers some handlers to be called when a new service is found
func SubscribeOnAddService(f servicediscovery.ServiceFunc) func(provider *KubeServiceProvider) *KubeServiceProvider {
	return func(provider *KubeServiceProvider) *KubeServiceProvider {
		provider.onAddServiceHandlers = append(provider.onAddServiceHandlers, f)
		return provider
	}
}

//SubscribeOnRemoveService registers some handlers to be called when a service is removed
func SubscribeOnRemoveService(f servicediscovery.ServiceFunc) func(provider *KubeServiceProvider) *KubeServiceProvider {
	return func(provider *KubeServiceProvider) *KubeServiceProvider {
		provider.onRemoveServiceHandlers = append(provider.onRemoveServiceHandlers, f)
		return provider
	}
}

//SubscribeOnUpdateService registers some handlers to be called when a service gets updated
func SubscribeOnUpdateService(f func(old servicediscovery.Service, new servicediscovery.Service)) func(provider *KubeServiceProvider) *KubeServiceProvider {
	return func(provider *KubeServiceProvider) *KubeServiceProvider {
		provider.onUpdateServiceHandlers = append(provider.onUpdateServiceHandlers, f)
		return provider
	}
}

//Compose composes provider functions
func Compose(funcs ...func(p *KubeServiceProvider) *KubeServiceProvider) func(p *KubeServiceProvider) *KubeServiceProvider {
	return func(p *KubeServiceProvider) *KubeServiceProvider {
		for _, f := range funcs {
			p = f(p)
		}
		return p
	}
}
