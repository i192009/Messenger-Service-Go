package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.zixel.cn/go/framework/k8smanager"
	"gitlab.zixel.cn/go/framework/xutil"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"gitlab.zixel.cn/go/framework/variant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
)

var k8s *kubernetes.Clientset

// 监视列表
var monitor = make(map[string][]string)
var informerFactory informers.SharedInformerFactory

var k8sManager = k8smanager.NewK8sClientManager()

// GetClientSet get k8sClient set
// get k8sClient set redundant logic, function ready for clean up
func getClientSet() (client *kubernetes.Clientset, err error) {
	if k8s != nil {
		return k8s, nil
	}

	var config *rest.Config
	configFile := os.Getenv("KUBERNETES_SERVICE_CONF")
	config = func() *rest.Config {
		if configFile == "" {
			return nil
		}

		configFile, err = xutil.ParsePath(configFile)
		if err != nil {
			return nil
		}

		fi, err := os.Stat(configFile)
		if err != nil {
			log.Println(err.Error())
			return nil
		}

		if fi.IsDir() {
			return nil
		}

		config, err = clientcmd.BuildConfigFromFlags("", configFile)
		if err != nil {
			log.Println(err.Error())
			return nil
		}

		return config
	}()

	if config == nil {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	// creates the clientset
	k8s, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k8s, nil
}

// getConfigMap Logic move to framework\k8sManager, function ready for clean up
func getConfigMap(ctx context.Context, namespace string, name string, key string) (data []byte, err error) {
	if _, err = getClientSet(); err != nil {
		return
	}

	configmap, err := k8s.CoreV1().ConfigMaps(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		return
	}

	val, ok := configmap.Data[key]
	if !ok {
		return
	}

	return []byte(val), nil
}

// Monitor redundant logic, function ready for clean up
func Monitor(namespace, name string, keys ...string) (err error) {
	informerFactory = informers.NewFilteredSharedInformerFactory(
		k8s,
		10*time.Minute,
		K8S_Namespace,
		func(opt *metaV1.ListOptions) {
			opt.FieldSelector = "metadata.name=" + name
		},
	)
	configMaps := informerFactory.Core().V1().ConfigMaps()
	configMaps.Informer().AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {

		},
		UpdateFunc: func(oldObj any, newObj any) {

		},
		DeleteFunc: func(obj any) {

		},
	})

	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
	return
}

// LoadConfigMap Logic move to framework\k8sManager, function ready for clean up
func LoadConfigMap(configEncrypted bool, data []byte, key string) (result variant.AbstractValue, err error) {
	log.Printf("LoadConfigMap: %s, encrypt=%t\n", key, configEncrypted)

	// 加密配置需要解密
	if configEncrypted {
		data = xutil.DecryptDataWithSn(data, GetString(xutil.LICENSE_SN, ""), GetString(xutil.LICENSE_PRIVATE_KEY, ""))
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

func Watch(configEncrypted bool, namespace string, name string, keys ...string) (err error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 服务结束则为context发送cancel
	go func() {
		for IsRunning {
			time.Sleep(time.Second)
		}

		cancel()
	}()

	// 将keys中没有在monitor[name]中出现的key添加到列表中
	if monitorKeys, ok := monitor[name]; ok {
		for i := range keys {
			ok = false
			for j := range monitorKeys {
				if keys[i] == monitorKeys[j] {
					ok = true
				}
			}

			if !ok {
				monitor[name] = append(monitor[name], keys[i])
			}
		}

		return
	}

	encryption := k8smanager.EncryptedConfig{IsEncrypted: configEncrypted}
	if configEncrypted {
		encryption.FixedPrivateKey = GetString(xutil.LICENSE_FIXED_PRIVATE_KEY, "")
	}

	for _, key := range keys {
		data, err := k8sManager.GetConfigMapData(namespace, name, key)
		if err != nil {
			log.Printf("k8s error get namespace %s, name:%s, key: %s\nerr= %s\n", namespace, name, key, err.Error())
			continue
		}

		// log.Printf("config map key = %s, data = \n%s\n", key, data)
		v, err := k8sManager.LoadConfigMap(encryption, data, key)
		if err != nil {
			log.Println(err.Error())
		}

		if v != nil {
			// 更新时需要加写锁
			func() {
				lock.Lock()
				defer lock.Unlock()

				o = variant.Merge(v, o)
			}()
		}
	}

	initConfigVariants()

	var w watch.Interface
	w, err = k8sManager.WatchConfigMap(ctx, namespace, name)
	if err != nil {
		return
	}

	monitor[name] = keys
	go func() {
		ch := w.ResultChan()
		for {
			select {
			case <-ctx.Done():
				return

			case evt := <-ch:
				switch t := evt.Object.(type) {
				case *v1.ConfigMap:
					if keys, ok := monitor[name]; ok {
						for _, key := range keys {
							// 先确认key存在
							c, ok := t.Data[key]
							if !ok {
								continue
							}

							v, err := k8sManager.LoadConfigMap(encryption, []byte(c), key)
							if err != nil {
								log.Println(err.Error())
							}

							if v != nil {
								// 更新时需要加写锁
								func() {
									lock.Lock()
									defer lock.Unlock()

									o = variant.Merge(v, o)
								}()

								log.Printf("ConfigMap update, namespace: %s, name: %s, key: %s\n", namespace, name, strings.Join(keys, ", "))
								// variant.Print(o, "  ", "  ")
								// 重新更新
								initConfigVariants()
							}
						}
					}
				default:
					if t != nil {
						log.Printf("got event type = %T\n", t)
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()

	return
}

//const (
//	PublicConfigName = "golang-config"
//)
//const (
//	KafkaCon = "kafka"
//	MongoCon = "mong_db"
//	MysqlCon = "mysql"
//	RedisCon = "redis"
//	MyConfig = "my_config"
//)
//
//var AllPublicConfigName = []string{KafkaCon, MongoCon, MysqlCon, RedisCon, MyConfig}
//var EventBus = bus.NewAsyncEventBus()
//
//type MysqlPublicConfig struct {
//	Url string `yaml:"url"`
//}
//
//type RedisPublicConfig struct {
//	Url      string `yaml:"url"`
//	Password string `yaml:"password"`
//	PoolSize int    `yaml:"pool_size"`
//}
//
//type KafkaPublicConfig struct {
//	Url string `yaml:"url"`
//}
//
//type MongoDBPublicConfig struct {
//	Url string `yaml:"url"`
//}
//
//type MysqlMyConfig struct {
//	Username string `yaml:"username" json:"username"`
//	Password string `yaml:"password" json:"password"`
//	DBName   string `yaml:"db_name" json:"db_name"`
//}
//
//type UsedPublicConfig struct {
//	ConfigNames []string `yaml:"config_name" json:"config_name"`
//}
//
//func (pConfig UsedPublicConfig) GetPublicConfigMap() map[string]interface{} {
//	if pConfig.ConfigNames == nil || len(pConfig.ConfigNames) <= 0 {
//		return nil
//	}
//	configDataMap := make(map[string]interface{})
//	for _, conName := range pConfig.ConfigNames {
//		switch conName {
//		case KafkaCon:
//			configDataMap[conName] = new(KafkaPublicConfig)
//		case MongoCon:
//			configDataMap[conName] = new(MongoDBPublicConfig)
//		case MysqlCon:
//			configDataMap[conName] = new(MysqlPublicConfig)
//		case RedisCon:
//			configDataMap[conName] = new(RedisPublicConfig)
//		}
//	}
//	return configDataMap
//}
//
//func LoadServiceConfigMap(ctx context.Context, namespace, ymlName, serviceName string, myConfigData interface{}) (err error,
//	kafkaConfig *KafkaPublicConfig, mysqlConfig *MysqlPublicConfig, redisConfig *RedisPublicConfig, mongoDBConfig *MongoDBPublicConfig, myConfig interface{}) {
//
//	configmap, err := k8sManager.GetConfigMap(namespace, serviceName)
//	if err != nil {
//		log.Printf("Failed to get configmap : %v \n", err)
//		return err, nil, nil, nil, nil, nil
//	}
//	datamap, ok := configmap.Data[ymlName]
//	if !ok {
//		log.Printf("Failed to get configmap : %v \n", err)
//		return err, nil, nil, nil, nil, nil
//	}
//
//	configData := &struct {
//		UsedPublicConfig UsedPublicConfig `yaml:"used_public_config" json:"used_public_config"`
//		MysqlMylConfig   MysqlMyConfig    `yaml:"mysql_config" json:"mysql_config"`
//	}{}
//	//处理config
//	transformMyConfig := func(datamap string, configData interface{}, myConfigData interface{}) error {
//		confBy, err := yaml2.YAMLToJSON([]byte(datamap))
//		if err != nil {
//			log.Printf(" LoadConfigMap yaml.YAMLToJSON: %v \n", err)
//			return err
//		}
//		err = json.Unmarshal(confBy, configData)
//		if err != nil {
//			log.Printf(" LoadConfigMap yaml.Unmarshal: %v \n", err)
//			return err
//		}
//		res := gjson.Get(string(confBy), MyConfig)
//		//myConf = []byte(res.String())
//		errB := json.Unmarshal([]byte(res.String()), myConfigData)
//		if errB != nil {
//			log.Printf("LoadConfigMap>>Marshal myConfigData is error,err:%s \n", errB)
//			return err
//		}
//		return nil
//	}
//	if err = transformMyConfig(datamap, configData, myConfigData); err != nil {
//		log.Printf("LoadConfigMap>> getMyConfig is error,err:%s \n", err)
//		return err, nil, nil, nil, nil, nil
//	}
//	myConfig = myConfigData
//	//获取公共配置
//	publicConfigMap := configData.UsedPublicConfig.GetPublicConfigMap()
//	if publicConfigMap != nil {
//
//		if publicConfigMap, err = initCommentConfigMap(namespace, PublicConfigName, publicConfigMap); err != nil {
//
//			log.Printf(" configmap InitCommentConfigMap: %v \n", err)
//			return err, nil, nil, nil, nil, nil
//		} else {
//			for _, cVaule := range publicConfigMap {
//				switch vaule := cVaule.(type) {
//				case *KafkaPublicConfig:
//					kafkaConfig = vaule
//				case *MongoDBPublicConfig:
//					mongoDBConfig = vaule
//				case *MysqlPublicConfig:
//					if len(vaule.Url) > 0 {
//						vaule.Url = strings.Replace(vaule.Url, "{username}", configData.MysqlMylConfig.Username, -1)
//						vaule.Url = strings.Replace(vaule.Url, "{password}", configData.MysqlMylConfig.Password, -1)
//						vaule.Url = strings.Replace(vaule.Url, "{db_name}", configData.MysqlMylConfig.DBName, -1)
//					}
//					mysqlConfig = vaule
//				case *RedisPublicConfig:
//					redisConfig = vaule
//				}
//			}
//		}
//	}
//	//监听配置变化，若变化 ，发布订阅事件
//	watchCh := make(chan map[string]interface{}, len(AllPublicConfigName)+3)
//	go func() {
//		w, err := k8sManager.WatchConfigMap(ctx, namespace, PublicConfigName)
//		if err != nil {
//			log.Printf(" configmap Watch: %v \n", err)
//			return
//		}
//		for {
//			select {
//			case e, _ := <-w.ResultChan():
//				obj := e.Object
//				// 转成config对象
//				cf := obj.(*v1.ConfigMap)
//				log.Printf("*********************** %s %s ***********************\n", e.Type, cf.Name)
//				if e.Type == watch.Added || e.Type == watch.Modified {
//					for _, conName := range AllPublicConfigName {
//						if _, ok := publicConfigMap[conName]; !ok {
//							continue
//						}
//						//如果堆积过多未消费，则剔除出最先进入得数据
//						if len(watchCh) > len(AllPublicConfigName) {
//							discard := <-watchCh
//							for k, v := range discard {
//								log.Printf("WatchConfigMap:configName %s  is discard,content:%s \n", k, v)
//							}
//						}
//						switch conName {
//						case KafkaCon:
//							kafkaConfig = new(KafkaPublicConfig)
//							decoder := yaml.NewDecoder(strings.NewReader(cf.Data[conName]))
//							if err = decoder.Decode(kafkaConfig); err != nil {
//								log.Printf(" Failed to get initCommentConfigMap configType%s, err: %v \n", cf.Data[conName], err)
//							} else {
//								conMap := make(map[string]interface{})
//								conMap[conName] = kafkaConfig
//								watchCh <- conMap
//								//ConfigEventBus.Publish(Kafka_Con, kafkaPublicConfig)
//							}
//						case MongoCon:
//							mongoDBConfig = new(MongoDBPublicConfig)
//							decoder := yaml.NewDecoder(strings.NewReader(cf.Data[conName]))
//							if err = decoder.Decode(mongoDBConfig); err != nil {
//								log.Printf(" Failed to get initCommentConfigMap configType%s, err: %v \n", cf.Data[conName], err)
//							} else {
//								conMap := make(map[string]interface{})
//								conMap[conName] = mongoDBConfig
//								watchCh <- conMap
//							}
//						case MysqlCon:
//							mysqlConfig = new(MysqlPublicConfig)
//							decoder := yaml.NewDecoder(strings.NewReader(cf.Data[conName]))
//							if err = decoder.Decode(mysqlConfig); err != nil {
//								log.Printf(" Failed to get initCommentConfigMap configType%s, err: %v \n", cf.Data[conName], err)
//							} else {
//								if len(mysqlConfig.Url) > 0 {
//									mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{username}", configData.MysqlMylConfig.Username, -1)
//									mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{password}", configData.MysqlMylConfig.Password, -1)
//									mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{db_name}", configData.MysqlMylConfig.DBName, -1)
//								}
//								conMap := make(map[string]interface{})
//								conMap[conName] = mysqlConfig
//								watchCh <- conMap
//							}
//						case RedisCon:
//							redisConfig = new(RedisPublicConfig)
//							decoder := yaml.NewDecoder(strings.NewReader(cf.Data[conName]))
//							if err = decoder.Decode(redisConfig); err != nil {
//								log.Printf(" Failed to get initCommentConfigMap configType%s, err: %v \n", cf.Data[conName], err)
//							} else {
//								conMap := make(map[string]interface{})
//								conMap[conName] = redisConfig
//								watchCh <- conMap
//							}
//						}
//					}
//
//				}
//			default:
//				time.Sleep(1 * time.Second)
//			}
//		}
//	}()
//	go func() {
//		w, err := k8sManager.WatchConfigMap(ctx, namespace, serviceName)
//		if err != nil {
//			log.Printf(" configmap Watch: %v \n", err)
//			return
//		}
//		for {
//			select {
//			case e := <-w.ResultChan():
//				obj := e.Object
//				// 转成config对象
//				cf := obj.(*v1.ConfigMap)
//				log.Printf("*********************** %s %s ***********************\n", e.Type, cf.Name)
//				if (e.Type == watch.Added || e.Type == watch.Modified) && cf.Name == serviceName {
//					if err = transformMyConfig(cf.Data[ymlName], configData, myConfigData); err != nil {
//						log.Printf("LoadConfigMap>> getMyConfig is error,err:%s \n", err)
//						continue
//					}
//					conMap := make(map[string]interface{})
//					conMap[MyConfig] = myConfigData
//					//如果堆积过多未消费，则剔除出最先进入得数据
//					if len(watchCh) > len(AllPublicConfigName) {
//						discard := <-watchCh
//						for k, v := range discard {
//							log.Printf("WatchConfigMap:configName %s  is discard,content:%s \n", k, v)
//						}
//					}
//					watchCh <- conMap
//					//关联更新
//					if mysqlConfig != nil {
//						if len(mysqlConfig.Url) > 0 {
//							mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{username}", configData.MysqlMylConfig.Username, -1)
//							mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{password}", configData.MysqlMylConfig.Password, -1)
//							mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{db_name}", configData.MysqlMylConfig.DBName, -1)
//						}
//						conMysqlMap := make(map[string]interface{})
//						conMysqlMap[MysqlCon] = mysqlConfig
//						watchCh <- conMysqlMap
//					}
//				}
//			default:
//				time.Sleep(1 * time.Second)
//			}
//		}
//	}()
//	go func() {
//		for {
//			conMap := <-watchCh
//			for k, v := range conMap {
//				EventBus.Publish(k, v)
//			}
//		}
//	}()
//	err = nil
//	return
//}
//
//func initCommentConfigMap(namespace, serviceName string, configDataMap map[string]interface{}) (cm map[string]interface{}, err error) {
//	if configDataMap == nil {
//		log.Printf("InitCommentConfigMap is nil \n")
//		err = errors.New("configDataMap is nil")
//		return
//	}
//
//	var configmap *v1.ConfigMap
//	configmap, err = k8sManager.GetConfigMap(namespace, serviceName)
//	if err != nil {
//		log.Printf("Failed to get initCommentConfigMap : %v \n", err)
//		return
//	}
//	for mkey, mvaule := range configDataMap {
//		if mvaule == nil {
//			continue
//		}
//
//		datamap, ok := configmap.Data[mkey]
//		if !ok {
//			log.Printf("Failed to get initCommentConfigMap : %v \n", err)
//			err = log.Errorf("key %s does not found", mkey)
//			return
//		}
//
//		log.Printf("initCommentConfigMap.Data:%s \n", datamap)
//		decoder := yaml.NewDecoder(strings.NewReader(datamap))
//		if err = decoder.Decode(mvaule); err != nil {
//			log.Printf(" Failed to get initCommentConfigMap configType%s, NewDecoder: %v \n", mvaule, err)
//			return
//		}
//	}
//
//	cm = configDataMap
//	return
//}
