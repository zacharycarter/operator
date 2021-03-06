package controller

import (
	"github.com/appscode/go/log"
	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	cs "github.com/kubedb/apimachinery/client/clientset/versioned/typed/kubedb/v1alpha1"
	amc "github.com/kubedb/apimachinery/pkg/controller"
	snapc "github.com/kubedb/apimachinery/pkg/controller/snapshot"
	esc "github.com/kubedb/elasticsearch/pkg/controller"
	mcc "github.com/kubedb/memcached/pkg/controller"
	mgc "github.com/kubedb/mongodb/pkg/controller"
	myc "github.com/kubedb/mysql/pkg/controller"
	pgc "github.com/kubedb/postgres/pkg/controller"
	rdc "github.com/kubedb/redis/pkg/controller"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
)

type Controller struct {
	amc.Config
	*amc.Controller
	promClient     pcm.MonitoringV1Interface
	cronController snapc.CronControllerInterface
	docker         Docker

	// DB controllers
	mgCtrl *mgc.Controller
	myCtrl *myc.Controller
	pgCtrl *pgc.Controller
	esCtrl *esc.Controller
	rdCtrl *rdc.Controller
	mcCtrl *mcc.Controller
}

type Docker struct {
	// docker Registry
	Registry string
	// Exporter tag
	ExporterTag string
}

func New(
	client kubernetes.Interface,
	apiExtKubeClient crd_cs.ApiextensionsV1beta1Interface,
	dbClient cs.KubedbV1alpha1Interface,
	promClient pcm.MonitoringV1Interface,
	cronController snapc.CronControllerInterface,
	docker Docker,
	opt amc.Config,
) *Controller {
	return &Controller{
		Controller: &amc.Controller{
			Client:           client,
			ExtClient:        dbClient,
			ApiExtKubeClient: apiExtKubeClient,
		},
		Config:         opt,
		docker:         docker,
		promClient:     promClient,
		cronController: cronController,
	}
}

// EnsureCustomResourceDefinitions ensures CRD for MySQl, DormantDatabase and Snapshot
func (c *Controller) EnsureCustomResourceDefinitions() error {
	log.Infoln("Ensuring CustomResourceDefinition...")
	crds := []*crd_api.CustomResourceDefinition{
		api.Elasticsearch{}.CustomResourceDefinition(),
		api.Postgres{}.CustomResourceDefinition(),
		api.MySQL{}.CustomResourceDefinition(),
		api.MongoDB{}.CustomResourceDefinition(),
		api.Redis{}.CustomResourceDefinition(),
		api.Memcached{}.CustomResourceDefinition(),
		api.DormantDatabase{}.CustomResourceDefinition(),
		api.Snapshot{}.CustomResourceDefinition(),
	}
	return apiext_util.RegisterCRDs(c.ApiExtKubeClient, crds)
}
