package k8s

import (
	"context"
	"encoding/json"
	syserror "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gitlab.zixel.cn/go/framework/config"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	yaml_k8s "k8s.io/apimachinery/pkg/util/yaml"

	yaml_v3 "gopkg.in/yaml.v3"
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

var pod_env_lock sync.Mutex

func GetPodsByServiceWithNamespace(ns string, name string, port int32) (result []PodInfos, err error) {
	if _, err = GetClientSet(); err != nil {
		log.Error("GetClientSet failed - ", err.Error())
		return
	}
	var service *v1.Service
	// Get kubernetes service list based on namespace
	service, err = k8s.CoreV1().Services(config.K8S_Namespace).Get(context.Background(), name, metaV1.GetOptions{})
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
	hostport := func() int {
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

	pods, err := k8s.CoreV1().Pods(ns).List(context.Background(), opt)

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
				Reason:             string(condition.Reason),
				Message:            condition.Message,
			})
		}

		result = append(result, PodInfos{
			Name:       pod.GetName(),
			Uuid:       string(pod.GetUID()),
			HostAddr:   pod.Status.HostIP,
			HostPort:   hostport,
			Conditions: conditions,
			Status:     string(pod.Status.Phase),
		})
	}

	return
}
func GetPodsByLabelsWithNamespace(ns string, labels map[string]string) (result []PodInfos, err error) {
	if _, err = GetClientSet(); err != nil {
		log.Error("GetClientSet failed - ", err.Error())
		return
	}
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
	options.LabelSelector = string(labelSelectorData.String())
	// Get kubernetes service list based on namespace
	pods, err := k8s.CoreV1().Pods(config.K8S_Namespace).List(context.Background(), options)
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
				Reason:             string(condition.Reason),
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
func DeletePodsByLabelsWithNamespace(ns string, labels map[string]string) (err error) {
	if _, err = GetClientSet(); err != nil {
		log.Error("GetClientSet failed - ", err.Error())
		return
	}
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
	options.LabelSelector = string(labelSelectorData.String())
	// Get kubernetes service list based on namespace
	err = k8s.CoreV1().Pods(config.K8S_Namespace).DeleteCollection(context.Background(), metaV1.DeleteOptions{}, options)
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
func SetupNewPodWithNamespace(ns string, name string, id string, filepath string) (err error) {
	if _, err = GetClientSet(); err != nil {
		log.Error("GetClientSet failed - ", err.Error())
		return
	}
	pod_env_lock.Lock()
	defer pod_env_lock.Unlock()
	// label 必须是 字符串 所以要用双引号包起来 否则在反序列化的时候会丢失
	os.Setenv("C_POD_NAME", name)
	os.Setenv("C_POD_ID", id)
	// os.Setenv("C_POD_REPLICAS", fmt.Sprintf("%v", replica))
	os.Setenv("C_POD_NAMESPACE", ns)
	os.Setenv("C_POD_OPS_TIME", time.Now().Format("2006-01-02_15.04.05"))
	// cmdstr := "eval \"echo \\\"$(cat ./deploy-model-compile.yaml)\\\"\""
	file, err := os.ReadFile(filepath)
	if err != nil {
		log.Error("read yaml file failed - ", err.Error())
		return
	}
	yamlStr := os.ExpandEnv(string(file))
	for _, v := range strings.Split(yamlStr, "---") {
		var deployment CommonDeploy
		err = yaml_v3.Unmarshal([]byte(v), &deployment)
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
			err = syserror.New("yaml is not a Pod setting")
			return
		}
		// YAML转JSON
		var deployJson []byte
		deployJson, err = yaml_k8s.ToJSON([]byte(v))
		if err != nil {
			log.Error("parse yaml to json failed:", err)
			return
		}

		// JSON转struct
		var pod v1.Pod
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
		_, err = k8s.CoreV1().Pods(ns).Create(context.Background(), &pod, metaV1.CreateOptions{})
		if statusError, isStatus := err.(*errors.StatusError); isStatus {
			if statusError.ErrStatus.Reason == metaV1.StatusReasonAlreadyExists {
				_, err = k8s.CoreV1().Pods(ns).Update(context.Background(), &pod, metaV1.UpdateOptions{})
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
func SetupNewPod(name string, id string, filepath string) (err error) {
	err = SetupNewPodWithNamespace(os.Getenv("PAAS_NAMESPACE"), name, id, filepath)
	return
}
func GetPodsByLabels(labels map[string]string) (result []PodInfos, err error) {
	result, err = GetPodsByLabelsWithNamespace(os.Getenv("PAAS_NAMESPACE"), labels)
	return
}

func DeletePodsByLabels(labels map[string]string) (err error) {
	err = DeletePodsByLabelsWithNamespace(os.Getenv("PAAS_NAMESPACE"), labels)
	return
}
func GetPodsByService(name string, port int32) (result []PodInfos, err error) {
	result, err = GetPodsByServiceWithNamespace(os.Getenv("PAAS_NAMESPACE"), name, port)
	return
}
func TransportRequest(pods []PodInfos, method, url string, body io.Reader, header http.Header) (res *http.Response, err error) {
	if len(pods) == 0 {
		err = syserror.New("pods size is empty")
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
func TransportRequestWithContainerName(pods []PodInfos, method, url string, body io.Reader, header http.Header, containerName string) (res *http.Response, err error) {
	if len(pods) == 0 {
		err = syserror.New("pods size is empty")
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
