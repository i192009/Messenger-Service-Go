package k8smanager

import (
	"context"
	"encoding/json"
	sysError "errors"
	"fmt"
	yamlV3 "gopkg.in/yaml.v3"
	"io"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	yamlK8s "k8s.io/apimachinery/pkg/util/yaml"
	"net/http"
	"os"
	"strings"
	"time"
)

type PodInfos struct {
	Name       string // 服务名
	Uuid       string // POD 实例ID
	HostAddr   string // POD 访问地址
	PodAddr    string // POD 内部地址
	PodPort    int    // POD 内部端口
	HostPort   int    // POD 访问端口
	Status     string
	Containers []containerInfo
	Conditions []PodCondition
}
type PodCondition struct {
	Type   string
	Status string
	// Last time we probed the condition.
	// +optional
	LastProbeTime time.Time
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime time.Time
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string
	// Human-readable message indicating details about last transition.
	// +optional
	Message string
}
type containerInfo struct {
	Name    string
	Image   string
	Command []string
	Ports   []ContainerPort
}
type ContainerPort struct {
	Name          string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	HostPort      int32  `json:"hostPort,omitempty" protobuf:"varint,2,opt,name=hostPort"`
	ContainerPort int32  `json:"containerPort" protobuf:"varint,3,opt,name=containerPort"`
	HostIP        string `json:"hostIP,omitempty" protobuf:"bytes,5,opt,name=hostIP"`
}

func (k *K8sClientManager) GetPod(namespaceName string, podName string) (*coreV1.Pod, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	pod, err := k.client.CoreV1().Pods(namespaceName).Get(context.Background(), podName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func (k *K8sClientManager) CreateNewPod(pod *coreV1.Pod) (*coreV1.Pod, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	createdPod, err := k.client.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metaV1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return createdPod, nil
}

func (k *K8sClientManager) DeletePod(namespaceName string, podName string) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	err := k.client.CoreV1().Pods(namespaceName).Delete(context.Background(), podName, metaV1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (k *K8sClientManager) ListPods(namespaceName string, options metaV1.ListOptions) (*coreV1.PodList, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	pods, err := k.client.CoreV1().Pods(namespaceName).List(context.Background(), options)
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func (k *K8sClientManager) CopyPods(sourceNamespace string, targetNamespace string, podNames []string) ([]*coreV1.Pod, error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	podList, err := k.client.CoreV1().Pods(sourceNamespace).List(context.Background(), metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	podNameSet := make(map[string]bool)
	for _, podName := range podNames {
		podNameSet[podName] = true
	}

	copiedPods := make([]*coreV1.Pod, 0)

	for _, pod := range podList.Items {
		if len(podNames) > 0 && !podNameSet[pod.Name] {
			// If specific pods are to be copied and this pod is not in the list, skip it
			continue
		}

		if len(pod.OwnerReferences) > 0 {
			// If the pod is owned by a ReplicaSet, DaemonSet, Job, etc., skip it
			continue
		}

		newPod := pod.DeepCopy()
		newPod.Namespace = targetNamespace
		newPod.ResourceVersion = "" // Clear the resource version
		newPod.UID = ""             // Clear the UID

		createdPod, err := k.client.CoreV1().Pods(targetNamespace).Create(context.Background(), newPod, metaV1.CreateOptions{})
		if err != nil {
			return nil, err
		}

		copiedPods = append(copiedPods, createdPod)
	}

	return copiedPods, nil
}

func (k *K8sClientManager) GetPodsByServiceWithNamespace(ns string, name string, port int32) (result []PodInfos, err error) {
	var service *coreV1.Service
	// Get kubernetes service list based on namespace
	service, err = k.client.CoreV1().Services(k.namespace).Get(context.Background(), name, metaV1.GetOptions{})
	// Logging error if failed to get service list
	if errors.IsNotFound(err) {
		log.Error("get services error - ", err.Error())
		return
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Error("get services error - ", statusError.ErrStatus.Message)
		return
	} else if err != nil {
		log.Error("get services error - ", err.Error())
		return
	}

	selector := service.Spec.Selector
	hostPort := func() int {
		for _, portInfo := range service.Spec.Ports {
			if portInfo.Port == port {
				return portInfo.TargetPort.IntValue()
			}
		}

		return 0
	}()

	// Setting set as selector
	set := labels.Set(selector)

	opt := metaV1.ListOptions{
		LabelSelector: set.AsSelector().String(),
	}

	pods, err := k.client.CoreV1().Pods(ns).List(context.Background(), opt)

	if err != nil {
		log.Error("list pod error - ", err.Error())
		return
	}

	for _, pod := range pods.Items {
		// if pod.Status.Phase != "Running" {
		// 	continue
		// }
		conditions := make([]PodCondition, 0, len(pod.Status.Conditions))
		for _, condition := range pod.Status.Conditions {
			conditions = append(conditions, PodCondition{
				Type:               string(condition.Type),
				Status:             string(condition.Status),
				LastProbeTime:      condition.LastProbeTime.Time,
				LastTransitionTime: condition.LastTransitionTime.Time,
				Reason:             condition.Reason,
				Message:            condition.Message,
			})
		}

		result = append(result, PodInfos{
			Name:       pod.GetName(),
			Uuid:       string(pod.GetUID()),
			HostAddr:   pod.Status.HostIP,
			HostPort:   hostPort,
			Conditions: conditions,
			Status:     string(pod.Status.Phase),
		})
	}

	return
}
func (k *K8sClientManager) GetPodsByLabels(labels map[string]string) (result []PodInfos, err error) {
	options := metaV1.ListOptions{}
	matchExpressions := make([]metaV1.LabelSelectorRequirement, 0, len(labels))
	for k, v := range labels {
		if v != "" {
			temp := metaV1.LabelSelectorRequirement{
				Key:      k,
				Operator: metaV1.LabelSelectorOpIn,
				Values: []string{
					v,
				},
			}
			matchExpressions = append(matchExpressions, temp)
		}
	}
	labelSelector := metaV1.LabelSelector{
		MatchExpressions: matchExpressions,
	}
	labelSelectorData, err := metaV1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		log.Error("set labelSelector failed - ", err.Error())
		return
	}
	options.LabelSelector = labelSelectorData.String()
	// Get kubernetes service list based on namespace
	pods, err := k.client.CoreV1().Pods(k.namespace).List(context.Background(), options)
	// Logging error if failed to get service list
	if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Error("get podList error - ", statusError.ErrStatus.Message)
		return
	} else if err != nil {
		log.Error("get podList error - ", err.Error())
		return
	}
	for _, pod := range pods.Items {
		// if pod.Status.Phase != "Running" {
		// 	continue
		// }
		conditions := make([]PodCondition, 0, len(pod.Status.Conditions))
		for _, condition := range pod.Status.Conditions {
			conditions = append(conditions, PodCondition{
				Type:               string(condition.Type),
				Status:             string(condition.Status),
				LastProbeTime:      condition.LastProbeTime.Time,
				LastTransitionTime: condition.LastTransitionTime.Time,
				Reason:             condition.Reason,
				Message:            condition.Message,
			})
		}
		containers := make([]containerInfo, 0, len(pod.Spec.Containers))
		for _, container := range pod.Spec.Containers {
			ports := make([]ContainerPort, 0, len(container.Ports))
			for _, port := range container.Ports {
				ports = append(ports, ContainerPort{
					Name:          port.Name,
					HostPort:      port.HostPort,
					ContainerPort: port.ContainerPort,
					HostIP:        port.HostIP,
				})
			}
			containers = append(containers, containerInfo{
				Name:    container.Name,
				Image:   container.Image,
				Command: container.Command,
				Ports:   ports,
			})
		}
		result = append(result, PodInfos{
			Name:       pod.GetName(),
			Uuid:       string(pod.GetUID()),
			HostAddr:   pod.Status.HostIP,
			HostPort:   int(pod.Spec.Containers[0].Ports[0].HostPort),
			PodAddr:    pod.Status.PodIP,
			PodPort:    int(pod.Spec.Containers[0].Ports[0].ContainerPort),
			Containers: containers,
			Conditions: conditions,
			Status:     string(pod.Status.Phase),
		})
	}
	return
}
func (k *K8sClientManager) DeletePodsByLabels(labels map[string]string) (err error) {
	options := metaV1.ListOptions{}
	matchExpressions := make([]metaV1.LabelSelectorRequirement, 0, len(labels))
	for k, v := range labels {
		if v != "" {
			temp := metaV1.LabelSelectorRequirement{
				Key:      k,
				Operator: metaV1.LabelSelectorOpIn,
				Values: []string{
					v,
				},
			}
			matchExpressions = append(matchExpressions, temp)
		}
	}
	labelSelector := metaV1.LabelSelector{
		MatchExpressions: matchExpressions,
	}
	labelSelectorData, err := metaV1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		log.Error("set labelSelector failed - ", err.Error())
		return
	}
	options.LabelSelector = labelSelectorData.String()
	// Get kubernetes service list based on namespace
	err = k.client.CoreV1().Pods(k.namespace).DeleteCollection(context.Background(), metaV1.DeleteOptions{}, options)
	// Logging error if failed to get service list
	if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Error("delete podList error - ", statusError.ErrStatus.Message)
		return
	} else if err != nil {
		log.Error("delete podList error - ", err.Error())
		return
	}
	return
}
func (k *K8sClientManager) SetupNewPodWithNamespace(ns string, name string, id string, filepath string) (err error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()
	// label 必须是 字符串 所以要用双引号包起来 否则在反序列化的时候会丢失
	_ = os.Setenv("C_POD_NAME", name)
	_ = os.Setenv("C_POD_ID", id)
	// os.Setenv("C_POD_REPLICAS", fmt.Sprintf("%v", replica))
	_ = os.Setenv("C_POD_NAMESPACE", ns)
	_ = os.Setenv("C_POD_OPS_TIME", time.Now().Format("2006-01-02_15.04.05"))
	// cmd str := "eval \"echo \\\"$(cat ./deploy-model-compile.yaml)\\\"\""
	file, err := os.ReadFile(filepath)
	if err != nil {
		log.Error("read yaml file failed - ", err.Error())
		return
	}
	yamlStr := os.ExpandEnv(string(file))
	for _, v := range strings.Split(yamlStr, "---") {
		var deployment CommonDeploy
		err = yamlV3.Unmarshal([]byte(v), &deployment)
		if err != nil {
			log.Error("parse pod yaml failed,err=", err)
			return
		}
		if deployment.Kind == "Pod" {
			log.Debug("Marshalled commonDeploy=", deployment)
			log.Debug("Marshalled commonDeploy=", deployment.Kind)
			log.Debug("Marshalled commonDeploy=", deployment.Metadata.Name)
			log.Debug("Marshalled commonDeploy=", deployment.Metadata.Namespace)
		} else {
			err = sysError.New("yaml is not a Pod setting")
			return
		}
		// YAML转JSON
		var deployJson []byte
		deployJson, err = yamlK8s.ToJSON([]byte(v))
		if err != nil {
			log.Error("parse yaml to json failed:", err)
			return
		}

		// JSON转struct
		var pod coreV1.Pod
		err = json.Unmarshal(deployJson, &pod)
		if err != nil {
			fmt.Println(err)
		}
		if pod.Kind == "Pod" {
			log.Debug("Marshalled Pod=", pod)
			log.Debug("Marshalled Pod=", pod.Kind)
			log.Debug("Marshalled Pod=", pod.Name)
			log.Debug("Marshalled Pod=", pod.Namespace)
		}
		_, err = k.client.CoreV1().Pods(ns).Create(context.Background(), &pod, metaV1.CreateOptions{})
		if statusError, isStatus := err.(*errors.StatusError); isStatus {
			if statusError.ErrStatus.Reason == metaV1.StatusReasonAlreadyExists {
				_, err = k.client.CoreV1().Pods(ns).Update(context.Background(), &pod, metaV1.UpdateOptions{})
				if statusError, isStatus := err.(*errors.StatusError); isStatus {
					log.Error("update Pod error - ", statusError.ErrStatus.Message)
					return
				} else if err != nil {
					log.Error("update Pod error - ", err.Error())
					return
				}
			} else {
				log.Error("create Pod error - ", statusError.ErrStatus.Message)
				return
			}
		} else if err != nil {
			log.Error("create Pod error - ", err.Error())
			return
		}
	}
	return
}
func (k *K8sClientManager) SetupNewPod(name string, id string, filepath string) (err error) {
	err = k.SetupNewPodWithNamespace(k.namespace, name, id, filepath)
	return
}
func (k *K8sClientManager) GetPodsByService(name string, port int32) (result []PodInfos, err error) {
	result, err = k.GetPodsByServiceWithNamespace(k.namespace, name, port)
	return
}
func (k *K8sClientManager) TransportRequest(pods []PodInfos, method, url string, body io.Reader, header http.Header) (res *http.Response, err error) {
	if len(pods) == 0 {
		err = sysError.New("pods size is empty")
		return
	}
	pod := pods[int(time.Now().Unix())%len(pods)]
	// res, err := http.Get(fmt.Sprintf("http://%v:%v%v", pod.Addr, pod.Port, c.Request.URL.Path))
	req, err := http.NewRequestWithContext(context.Background(), strings.ToUpper(method), fmt.Sprintf("http://%v:%v%v", pod.PodAddr, pod.PodPort, url), body)
	// req, err := http.NewRequestWithContext(context.Background(), strings.ToUpper(method), fmt.Sprintf("http://127.0.0.1:8743%v", url), body)
	req.Header = header
	if err != nil {
		return
	}
	res, err = http.DefaultClient.Do(req)
	return
}
func (k *K8sClientManager) TransportRequestWithContainerName(pods []PodInfos, method, url string, body io.Reader, header http.Header, containerName string) (res *http.Response, err error) {
	if len(pods) == 0 {
		err = sysError.New("pods size is empty")
		return
	}
	pod := pods[int(time.Now().Unix())%len(pods)]
	port := pod.PodPort
	if containerName != "" {
		for _, container := range pod.Containers {
			if container.Name == containerName {
				port = int(container.Ports[0].ContainerPort)
			}
		}
	}
	// res, err := http.Get(fmt.Sprintf("http://%v:%v%v", pod.Addr, pod.Port, c.Request.URL.Path))
	req, err := http.NewRequestWithContext(context.Background(), strings.ToUpper(method), fmt.Sprintf("http://%v:%v%v", pod.PodAddr, port, url), body)
	// req, err := http.NewRequestWithContext(context.Background(), strings.ToUpper(method), fmt.Sprintf("http://127.0.0.1:8743%v", url), body)
	req.Header = header
	if err != nil {
		return
	}
	res, err = http.DefaultClient.Do(req)
	return
}
