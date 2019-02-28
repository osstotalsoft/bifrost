package kubernetes

import (
	"github.com/osstotalsoft/bifrost/servicediscovery"
	log "github.com/sirupsen/logrus"
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
	"time"
)

type Provider struct {
	onAddServiceHandlers    []servicediscovery.ServiceFunc
	onRemoveServiceHandlers []servicediscovery.ServiceFunc
	onUpdateServiceHandlers []func(old servicediscovery.Service, new servicediscovery.Service)
	stop                    chan struct{}
	clientset               *kubernetes.Clientset
}

const ResourceLabelName = "api-gateway/resource"
const SecuredLabelName = "api-gateway/secured"

func NewKubernetesServiceDiscoveryProvider(inCluster bool) *Provider {

	var config *rest.Config
	var err error

	if inCluster {
		config, err = rest.InClusterConfig()
	} else {
		config, err = outOfClusterConfig()
	}

	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return &Provider{
		onAddServiceHandlers:    []servicediscovery.ServiceFunc{},
		onRemoveServiceHandlers: []servicediscovery.ServiceFunc{},
		onUpdateServiceHandlers: []func(old servicediscovery.Service, new servicediscovery.Service){},
		clientset:               clientset,
		stop:                    make(chan struct{}),
	}
}

func outOfClusterConfig() (*rest.Config, error) {
	kubeconfig := filepath.Join(os.Getenv("USERPROFILE"), ".kube", "config")

	// use the current context in kubeconfig
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func Start(provider *Provider) *Provider {

	watchlist := newServicesListWatch(provider.clientset.CoreV1().RESTClient())

	_, controller := cache.NewInformer(watchlist, &corev1.Service{}, time.Second*0, cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc(provider),
		DeleteFunc: deleteFunc(provider),
		UpdateFunc: updateFunc(provider),
	})
	go controller.Run(provider.stop)

	return provider
}

func updateFunc(provider *Provider) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		oldSrv := oldObj.(*corev1.Service)
		newSrv := newObj.(*corev1.Service)
		log.Debugf("KubernetesProvider: Service updated old: %s, new: %s", oldSrv.String(), newSrv.String())

		callUpdateSubscribers(provider.onUpdateServiceHandlers, mapToService(oldSrv), mapToService(newSrv))
	}
}

func deleteFunc(provider *Provider) func(obj interface{}) {
	return func(obj interface{}) {
		srv := obj.(*corev1.Service)
		log.Debugf("KubernetesProvider: Service deleted: %s", srv)

		callSubscribers(provider.onRemoveServiceHandlers, mapToService(srv))
	}
}

func addFunc(provider *Provider) func(obj interface{}) {
	return func(obj interface{}) {
		srv := obj.(*corev1.Service)
		log.Debugf("KubernetesProvider: New service : %v", srv)

		callSubscribers(provider.onAddServiceHandlers, mapToService(srv))
	}
}

func mapToService(srv *corev1.Service) servicediscovery.Service {
	secured, _ := strconv.ParseBool(srv.Labels[SecuredLabelName])

	return servicediscovery.Service{
		Address:   "http://" + srv.Name + "." + srv.Namespace, // "http://kube-worker1:32344",
		Version:   srv.ResourceVersion,
		UID:       string(srv.UID),
		Name:      srv.Name,
		Resource:  srv.Labels[ResourceLabelName], // "api1",
		Secured:   secured,
		Namespace: srv.Namespace, // "gateway",
	}
}

// to apply modification to ListOptions with a field selector, a label selector, or any other desired options.
func newServicesListWatch(c cache.Getter) *cache.ListWatch {
	resource := "services"

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = ResourceLabelName
		return c.Get().
			//Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Do().
			Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.LabelSelector = ResourceLabelName
		return c.Get().
			//Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Watch()
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

func Stop(provider *Provider) *Provider {
	close(provider.stop)
	return provider
}

func SubscribeOnAddService(f servicediscovery.ServiceFunc) func(provider *Provider) *Provider {
	return func(provider *Provider) *Provider {
		provider.onAddServiceHandlers = append(provider.onAddServiceHandlers, f)
		return provider
	}
}

func SubscribeOnRemoveService(f servicediscovery.ServiceFunc) func(provider *Provider) *Provider {
	return func(provider *Provider) *Provider {
		provider.onRemoveServiceHandlers = append(provider.onRemoveServiceHandlers, f)
		return provider
	}
}

func SubscribeOnUpdateService(f func(old servicediscovery.Service, new servicediscovery.Service)) func(provider *Provider) *Provider {
	return func(provider *Provider) *Provider {
		provider.onUpdateServiceHandlers = append(provider.onUpdateServiceHandlers, f)
		return provider
	}
}

func Compose(funcs ...func(p *Provider) *Provider) func(p *Provider) *Provider {
	return func(p *Provider) *Provider {
		for _, f := range funcs {
			p = f(p)
		}
		return p
	}
}
