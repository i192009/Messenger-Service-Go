package k8smanager

import (
	"os"
	"sync"

	"gitlab.zixel.cn/go/framework/logger"
	"gitlab.zixel.cn/go/framework/xutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var log = logger.Get()

type K8sClientManager struct {
	client                *kubernetes.Clientset
	namespace             string
	mutex                 sync.Mutex
	configMapListenStatus map[string]chan ConfigmapListeningObject
	configMapListenMutex  sync.Mutex
}

func NewK8sClientManager() (k8sClientManager *K8sClientManager) {
	var k8sConfig *rest.Config
	var err error

	k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		kubeConfig := os.Getenv("KUBERNETES_SERVICE_CONF")

		if kubeConfig == "" {
			kubeConfig = "~/.kube/config"
		}

		if kubeConfig[0] == '~' {
			kubeConfig, err = xutil.ParsePath(kubeConfig)
			if err != nil {
				log.Errorf("Error expanding home: %s", err.Error())
				return nil
			}
		}

		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			log.Errorf("Error building kube config: %s", err.Error())
			return nil
		}
	} else {
		log.Info("Kubernetes client using in cluster config")
	}

	client, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Errorf("Error building kubernetes client set: %s", err.Error())
		return nil
	}

	k8sClientManager = &K8sClientManager{
		client:                client,
		namespace:             os.Getenv("PAAS_NAMESPACE"),
		mutex:                 sync.Mutex{},
		configMapListenStatus: make(map[string]chan ConfigmapListeningObject),
		configMapListenMutex:  sync.Mutex{},
	}

	return
}
