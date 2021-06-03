package extension

import (
	log "github.com/sirupsen/logrus"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj-labs/argocd-extensions/pkg/api"
)

type Opts func(ctrl *extensionController)

func NewController(
	client dynamic.NamespaceableResourceInterface,
	informer cache.SharedIndexInformer,
	apiFactory api.Factory,
	opts ...Opts,
) *extensionController {
	ctrl := &extensionController{
		client:          client,
		informer:        informer,
		apiFactory:      apiFactory,
	}
	for i := range opts {
		opts[i](ctrl)
	}
	return ctrl
}

func (c *extensionController) Run(threadiness int, stopCh <-chan struct{}) {
	defer runtimeutil.HandleCrash()

	log.Warn("Controller is running.")
	for i := 0; i < threadiness; i++ {

	}
	<-stopCh
	log.Warn("Controller has stopped.")
}

type extensionController struct {
	client            dynamic.NamespaceableResourceInterface
	informer          cache.SharedIndexInformer
	apiFactory        api.Factory
}