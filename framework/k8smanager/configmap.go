package k8smanager

import (
	"context"
	"fmt"
	"gitlab.zixel.cn/go/framework/variant"
	"gitlab.zixel.cn/go/framework/xutil"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"path/filepath"
	"time"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConfigMapCreateEvent = iota
	ConfigMapUpdateEvent
	ConfigMapDeleteEvent
	ConfigMapStoppedEvent
)

type ConfigmapListeningObject struct {
	Configmap *coreV1.ConfigMap
	EventType int
}

type EncryptedConfig struct {
	IsEncrypted     bool
	FixedPrivateKey string
}

func (k *K8sClientManager) GetConfigMap(namespaceName string, configMapName string) (*coreV1.ConfigMap, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	configMap, err := k.client.CoreV1().ConfigMaps(namespaceName).Get(context.Background(), configMapName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return configMap, nil
}

func (k *K8sClientManager) GetConfigMapData(namespaceName string, configMapName string, key string) (data []byte, err error) {
	configmap, err := k.GetConfigMap(namespaceName, configMapName)
	if err != nil {
		return
	}

	val, ok := configmap.Data[key]
	if !ok {
		return
	}

	return []byte(val), nil
}

func (k *K8sClientManager) LoadConfigMap(configEncrypted EncryptedConfig, data []byte, key string) (result variant.AbstractValue, err error) {
	// 加密配置需要解密
	if configEncrypted.IsEncrypted {
		data = xutil.DecryptDataWithFixedPrivateKey(data, configEncrypted.FixedPrivateKey)
	}
	// 根据key的名字决定使用哪个Parser
	switch filepath.Ext(key) {
	case ".conf", ".json":
		result, err = variant.LoadJson(data)
		if err != nil {
			return
		}

	case ".yaml", ".yml":
		result, err = variant.LoadYaml(data)
		if err != nil {
			return
		}
	}

	return
}

func (k *K8sClientManager) CreateConfigMap(namespaceName string, configMap *coreV1.ConfigMap) (*coreV1.ConfigMap, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	configMap, err := k.client.CoreV1().ConfigMaps(namespaceName).Create(context.Background(), configMap, metaV1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return configMap, nil
}

func (k *K8sClientManager) UpdateConfigMap(namespaceName string, configMap *coreV1.ConfigMap) (*coreV1.ConfigMap, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	configMap, err := k.client.CoreV1().ConfigMaps(namespaceName).Update(context.Background(), configMap, metaV1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return configMap, nil
}

func (k *K8sClientManager) DeleteConfigMap(namespaceName string, configMapName string) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	err := k.client.CoreV1().ConfigMaps(namespaceName).Delete(context.Background(), configMapName, metaV1.DeleteOptions{})

	return err
}

func (k *K8sClientManager) ListConfigMaps(namespaceName string, options metaV1.ListOptions) (*coreV1.ConfigMapList, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	configMaps, err := k.client.CoreV1().ConfigMaps(namespaceName).List(context.Background(), options)
	if err != nil {
		return nil, err
	}

	return configMaps, nil
}

func (k *K8sClientManager) WatchConfigMap(ctx context.Context, namespaceName string, name string) (watch.Interface, error) {
	w, err := k.client.CoreV1().ConfigMaps(namespaceName).Watch(ctx, metaV1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})

	if err != nil {
		return nil, err
	}

	return w, nil
}

func (k *K8sClientManager) CopyConfigMap(namespaceName string, configMapName string, newNamespace string) (*coreV1.ConfigMap, error) {
	configMap := &coreV1.ConfigMap{}
	var err error

	if configMap, err = k.GetConfigMap(namespaceName, configMapName); err != nil {
		return nil, err
	}

	configMap.ObjectMeta.Namespace = newNamespace
	configMap.ResourceVersion = ""

	if configMap, err = k.CreateConfigMap(newNamespace, configMap); err != nil {
		return nil, err
	}

	return configMap, nil
}

func (k *K8sClientManager) ListenConfigMap(namespaceName string, configMapName string) (chan ConfigmapListeningObject, error) {
	configMapChan := make(chan ConfigmapListeningObject, 20)

	factory := informers.NewSharedInformerFactory(k.client, 0)
	configMapInformer := factory.Core().V1().ConfigMaps().Informer()

	_, err := configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			configMap := obj.(*coreV1.ConfigMap)
			if configMap.Namespace == namespaceName && configMap.Name == configMapName {
				configmapListeningObject := ConfigmapListeningObject{
					Configmap: configMap,
					EventType: ConfigMapCreateEvent,
				}
				configMapChan <- configmapListeningObject
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newConfigMap := newObj.(*coreV1.ConfigMap)
			if newConfigMap.Namespace == namespaceName && newConfigMap.Name == configMapName {
				configmapListeningObject := ConfigmapListeningObject{
					Configmap: newConfigMap,
					EventType: ConfigMapUpdateEvent,
				}
				configMapChan <- configmapListeningObject
			}
		},
		DeleteFunc: func(obj interface{}) {
			configMap := obj.(*coreV1.ConfigMap)
			if configMap.Namespace == namespaceName && configMap.Name == configMapName {
				configmapListeningObject := ConfigmapListeningObject{
					Configmap: configMap,
					EventType: ConfigMapDeleteEvent,
				}
				configMapChan <- configmapListeningObject
			}
		},
	})

	if err != nil {
		return nil, err
	}

	stopCh := make(chan struct{})

	k.configMapListenMutex.Lock()
	defer k.configMapListenMutex.Unlock()
	key := fmt.Sprintf("%s:%s", namespaceName, configMapName)
	k.configMapListenStatus[key] = configMapChan
	go func() {
		var abort = false
		for !abort {
			time.Sleep(1 * time.Second)

			abort = func() bool {
				k.configMapListenMutex.Lock()
				defer k.configMapListenMutex.Unlock()
				if _, ok := k.configMapListenStatus[key]; !ok {
					close(stopCh)
					return true
				}

				return false
			}()
		}
	}()

	go configMapInformer.Run(stopCh)

	return configMapChan, nil
}

func (k *K8sClientManager) StopListenConfigMap(namespaceName string, configMapName string) {
	k.configMapListenMutex.Lock()
	defer k.configMapListenMutex.Unlock()
	key := fmt.Sprintf("%s:%s", namespaceName, configMapName)
	c := k.configMapListenStatus[key]
	delete(k.configMapListenStatus, key)

	c <- ConfigmapListeningObject{
		EventType: ConfigMapStoppedEvent,
	}
	close(c)
}
