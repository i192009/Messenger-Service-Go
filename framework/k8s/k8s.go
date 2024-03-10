package k8s

import (
	"gitlab.zixel.cn/go/framework/logger"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

/*
* Package k8s, Logic move to k8sManager, k8s package ready for clean up
 */

var k8s *kubernetes.Clientset
var log = logger.Get()

type CommonDeploy struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	}
}

// GetClientSet get k8sClient set
func GetClientSet() (cclientset *kubernetes.Clientset, err error) {
	if k8s != nil {
		return k8s, nil
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	k8s, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k8s, nil
}
