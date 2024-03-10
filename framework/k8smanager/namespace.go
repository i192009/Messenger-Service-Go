package k8smanager

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *K8sClientManager) GetNamespace(namespaceName string) (*corev1.Namespace, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	namespace, err := k.client.CoreV1().Namespaces().Get(context.Background(), namespaceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return namespace, nil
}

func (k *K8sClientManager) CreateNamespace(namespace *corev1.Namespace) (*corev1.Namespace, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	createdNamespace, err := k.client.CoreV1().Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return createdNamespace, nil
}

func (k *K8sClientManager) UpdateNamespace(namespace *corev1.Namespace) (*corev1.Namespace, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	updatedNamespace, err := k.client.CoreV1().Namespaces().Update(context.Background(), namespace, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return updatedNamespace, nil
}

func (k *K8sClientManager) DeleteNamespace(namespaceName string) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	err := k.client.CoreV1().Namespaces().Delete(context.Background(), namespaceName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
