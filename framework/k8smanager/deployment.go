package k8smanager

import (
	"context"
	"encoding/json"
	sysError "errors"
	"fmt"
	yamlV3 "gopkg.in/yaml.v3"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	yamlK8s "k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"strings"
	"time"
)

type CommonDeploy struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	}
}

type DeploymentMetaData struct {
	Name                       string            `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	GenerateName               string            `json:"generateName,omitempty" protobuf:"bytes,2,opt,name=generateName"`
	Namespace                  string            `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	UID                        types.UID         `json:"uid,omitempty" protobuf:"bytes,5,opt,name=uid,casttype=k8s.io/kubernetes/pkg/types.UID"`
	ResourceVersion            string            `json:"resourceVersion,omitempty" protobuf:"bytes,6,opt,name=resourceVersion"`
	Generation                 int64             `json:"generation,omitempty" protobuf:"varint,7,opt,name=generation"`
	CreationTimestamp          time.Time         `json:"creationTimestamp,omitempty" protobuf:"bytes,8,opt,name=creationTimestamp"`
	DeletionTimestamp          *time.Time        `json:"deletionTimestamp,omitempty" protobuf:"bytes,9,opt,name=deletionTimestamp"`
	DeletionGracePeriodSeconds *int64            `json:"deletionGracePeriodSeconds,omitempty" protobuf:"varint,10,opt,name=deletionGracePeriodSeconds"`
	Labels                     map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
	Annotations                map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`
	Finalizers                 []string          `json:"finalizers,omitempty" patchStrategy:"merge" protobuf:"bytes,14,rep,name=finalizers"`
}

func (k *K8sClientManager) GetDeploymentsMetadataByLabelsWithNamespace(ns string, labels map[string]string, checkReady bool) (result []DeploymentMetaData, err error) {
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
	deployments, err := k.client.AppsV1().Deployments(ns).List(context.Background(), options)
	// Logging error if failed to get service list
	if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Error("get deployment error - ", statusError.ErrStatus.Message)
		return
	}
	if err != nil {
		log.Error("get deployment error - ", err.Error())
		return
	}
	for _, deployment := range deployments.Items {
		if deployment.Status.Replicas == deployment.Status.ReadyReplicas || !checkReady {
			tmp := DeploymentMetaData{
				Name:                       deployment.Name,
				GenerateName:               deployment.GenerateName,
				Namespace:                  deployment.Namespace,
				UID:                        deployment.UID,
				ResourceVersion:            deployment.ResourceVersion,
				Generation:                 deployment.Generation,
				CreationTimestamp:          deployment.CreationTimestamp.Time,
				DeletionGracePeriodSeconds: deployment.DeletionGracePeriodSeconds,
				Labels:                     deployment.Labels,
				Annotations:                deployment.Annotations,
				Finalizers:                 deployment.Finalizers,
			}
			if deployment.DeletionTimestamp != nil {
				tmp.DeletionTimestamp = &deployment.DeletionTimestamp.Time
			}
			result = append(result, tmp)
		}
	}
	return
}

func (k *K8sClientManager) GetDeploymentsMetadataByLabels(labels map[string]string) (result []DeploymentMetaData, err error) {
	result, err = k.GetDeploymentsMetadataByLabelsWithNamespace(k.namespace, labels, false)
	return
}
func (k *K8sClientManager) GetReadyDeploymentsMetadataByLabels(labels map[string]string) (result []DeploymentMetaData, err error) {
	result, err = k.GetDeploymentsMetadataByLabelsWithNamespace(k.namespace, labels, true)
	return
}

func (k *K8sClientManager) SetupNewDeploymentByFile(name string, id string, replica int64, filepath string) (err error) {
	err = k.SetupNewDeploymentWithNamespaceAndParamsByFile(k.namespace, name, id, make(map[string]string), replica, filepath)
	return
}
func (k *K8sClientManager) SetupNewDeploymentAndParamsByFile(name string, id string, params map[string]string, replica int64, filepath string) (err error) {
	err = k.SetupNewDeploymentWithNamespaceAndParamsByFile(k.namespace, name, id, params, replica, filepath)
	return
}
func (k *K8sClientManager) SetupNewDeploymentByValue(name string, id string, replica int64, value *string) (err error) {
	err = k.SetupNewDeploymentWithNamespaceAndParams(k.namespace, name, id, make(map[string]string), replica, value)
	return
}
func (k *K8sClientManager) SetupNewDeploymentAndParamsByValue(name string, id string, params map[string]string, replica int64, value *string) (err error) {
	err = k.SetupNewDeploymentWithNamespaceAndParams(k.namespace, name, id, params, replica, value)
	return
}
func (k *K8sClientManager) DeleteDeploymentsByLabels(labels map[string]string) (err error) {
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
	err = k.client.AppsV1().Deployments(k.namespace).DeleteCollection(context.Background(), metaV1.DeleteOptions{}, options)
	// Logging error if failed to get service list
	if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Error("delete deployment error - ", statusError.ErrStatus.Message)
		return
	} else if err != nil {
		log.Error("delete deployment error - ", err.Error())
		return
	}
	return
}
func (k *K8sClientManager) SetupNewDeploymentWithNamespaceAndParamsByFile(ns string, name string, id string, params map[string]string, replica int64, filepath string) (err error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		log.Error("read yaml file failed - ", err.Error())
		return
	}
	fileStr := string(file)
	err = k.SetupNewDeploymentWithNamespaceAndParams(ns, name, id, params, replica, &fileStr)
	return
}

func (k *K8sClientManager) SetupNewDeploymentWithNamespaceAndParams(ns string, name string, id string, params map[string]string, replica int64, value *string) (err error) {
	if value == nil || *value == "" {
		err = sysError.New("value is empty")
		return
	}
	k.mutex.Lock()
	defer k.mutex.Unlock()
	// label 必须是 字符串 所以要用双引号包起来 否则在反序列化的时候会丢失
	_ = os.Setenv("C_DEPL_NAME", name)
	_ = os.Setenv("C_DEPL_ID", id)
	_ = os.Setenv("C_DEPL_REPLICAS", fmt.Sprintf("%v", replica))
	_ = os.Setenv("C_DEPL_NAMESPACE", ns)
	_ = os.Setenv("C_DEPL_OPS_TIME", time.Now().Format("2006-01-02_15.04.05"))
	for k, v := range params {
		_ = os.Setenv(k, fmt.Sprintf("%v", v))
	}
	yamlStr := os.ExpandEnv(*value)
	for _, v := range strings.Split(yamlStr, "---") {
		var deployment CommonDeploy
		err = yamlV3.Unmarshal([]byte(v), &deployment)
		if err != nil {
			log.Error("parse pod yaml failed,err=", err)
			return
		}
		if deployment.Kind == "Deployment" {
			log.Debug("Marshalled commonDeploy=", deployment)
			log.Debug("Marshalled commonDeploy=", deployment.Kind)
			log.Debug("Marshalled commonDeploy=", deployment.Metadata.Name)
			log.Debug("Marshalled commonDeploy=", deployment.Metadata.Namespace)
		} else {
			err = sysError.New("yaml is not a Deployment setting")
			return
		}

		var deployJson []byte
		deployJson, err = yamlK8s.ToJSON([]byte(v))
		if err != nil {
			log.Error("parse yaml to json failed:", err)
			return
		}

		// JSON转struct
		var deploy v1.Deployment
		err = json.Unmarshal(deployJson, &deploy)
		if err != nil {
			fmt.Println(err)
		}
		if deploy.Kind == "Deployment" {
			log.Debug("Marshalled Deployment=", deploy)
			log.Debug("Marshalled Deployment=", deploy.Kind)
			log.Debug("Marshalled Deployment=", deploy.Name)
			log.Debug("Marshalled Deployment=", deploy.Namespace)
		}
		_, err = k.client.AppsV1().Deployments(ns).Create(context.Background(), &deploy, metaV1.CreateOptions{})
		if statusError, isStatus := err.(*errors.StatusError); isStatus {
			if statusError.ErrStatus.Reason == metaV1.StatusReasonAlreadyExists {
				_, err = k.client.AppsV1().Deployments(ns).Update(context.Background(), &deploy, metaV1.UpdateOptions{})
				if statusError, isStatus := err.(*errors.StatusError); isStatus {
					log.Error("update deployment error - ", statusError.ErrStatus.Message)
					return
				} else if err != nil {
					log.Error("update deployment error - ", err.Error())
					return
				}
			} else {
				log.Error("create deployment error - ", statusError.ErrStatus.Message)
				return
			}
		} else if err != nil {
			log.Error("create deployment error - ", err.Error())
			return
		}
	}
	return
}
