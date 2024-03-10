package k8smanager

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *K8sClientManager) GetService(namespaceName string, serviceName string) (*corev1.Service, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	service, err := k.client.CoreV1().Services(namespaceName).Get(context.Background(), serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (k *K8sClientManager) CreateService(service *corev1.Service) (*corev1.Service, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	createdService, err := k.client.CoreV1().Services(service.Namespace).Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return createdService, nil
}

func (k *K8sClientManager) DeleteService(namespaceName string, serviceName string) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	err := k.client.CoreV1().Services(namespaceName).Delete(context.Background(), serviceName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
