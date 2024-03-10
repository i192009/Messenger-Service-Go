package configMap

import (
	"context"
	"encoding/json"
	"fmt"
	"gitlab.zixel.cn/go/framework/k8smanager"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"gitlab.zixel.cn/go/framework/bus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	yaml2 "github.com/ghodss/yaml"
	"gopkg.in/yaml.v2"
)

const (
	PUBILC_CONFIG_NAME = "golang-config"
)
const (
	Kafka_Con = "kafka"
	Mong_Con  = "mong_db"
	Mysql_Con = "mysql"
	Redis_Con = "redis"
	MY_CONFIG = "my_config"
)

var k8sManager = k8smanager.NewK8sClientManager()
var AllPublicConfigName = []string{Kafka_Con, Mong_Con, Mysql_Con, Redis_Con, MY_CONFIG}
var ConfigEventBus = bus.NewAsyncEventBus()

type MysqlPublicConfig struct {
	Url string `yaml:"url"`
}

type RedisPublicConfig struct {
	Url      string `yaml:"url"`
	Password string `yaml:"password"`
	PoolSize int    `yaml:"pool_size"`
}

type KafkaPublicConfig struct {
	Url string `yaml:"url"`
}

type MongoDBPublicConfig struct {
	Url string `yaml:"url"`
}

type MysqlMyConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	DBName   string `yaml:"db_name" json:"db_name"`
}

//type MongoDBMyConfig struct {
//	Username string `yaml:"username"`
//	Password string `yaml:"password"`
//}

type UsedPublicConfig struct {
	ConfigNames []string `yaml:"config_name" json:"config_name"`
}

func (pConfig UsedPublicConfig) GetPublicConfigMap() map[string]interface{} {
	if pConfig.ConfigNames == nil || len(pConfig.ConfigNames) <= 0 {
		return nil
	}
	configDataMap := make(map[string]interface{})
	for _, conName := range pConfig.ConfigNames {
		switch conName {
		case Kafka_Con:
			configDataMap[conName] = new(KafkaPublicConfig)
		case Mong_Con:
			configDataMap[conName] = new(MongoDBPublicConfig)
		case Mysql_Con:
			configDataMap[conName] = new(MysqlPublicConfig)
		case Redis_Con:
			configDataMap[conName] = new(RedisPublicConfig)
		}
	}
	return configDataMap
}

func initCommentConfigMap(ctx context.Context, namespace, serviceName string, configDataMap map[string]interface{}) (cm map[string]interface{}, err error) {
	if configDataMap == nil {
		fmt.Printf("InitCommentConfigMap is nil \n")
		err = errors.New("configDataMap is nil")
		return
	}

	var configmap *v1.ConfigMap
	configmap, err = k8sManager.GetConfigMap(namespace, serviceName)
	if err != nil {
		fmt.Printf("Failed to get initCommentConfigMap : %v \n", err)
		return
	}
	for mkey, mvaule := range configDataMap {
		if mvaule == nil {
			continue
		}

		datamap, ok := configmap.Data[mkey]
		if !ok {
			fmt.Printf("Failed to get initCommentConfigMap : %v \n", err)
			err = fmt.Errorf("key %s does not found", mkey)
			return
		}

		fmt.Printf("initCommentConfigMap.Data:%s \n", datamap)
		decoder := yaml.NewDecoder(strings.NewReader(datamap))
		if err = decoder.Decode(mvaule); err != nil {
			fmt.Printf(" Failed to get initCommentConfigMap configType%s, NewDecoder: %v \n", mvaule, err)
			return
		}
	}

	cm = configDataMap
	return
}

func LoadServiceConfigMap(ctx context.Context, namespace, ymlName, serviceName string, myConfigData interface{}) (err error,
	kafkaConfig *KafkaPublicConfig, mysqlConfig *MysqlPublicConfig, redisConfig *RedisPublicConfig, mongoDBConfig *MongoDBPublicConfig, myConfig interface{}) {

	configmap, err := k8sManager.GetConfigMap(namespace, serviceName)
	if err != nil {
		fmt.Printf("Failed to get configmap : %v \n", err)
		return err, nil, nil, nil, nil, nil
	}
	datamap, ok := configmap.Data[ymlName]
	if !ok {
		fmt.Printf("Failed to get configmap : %v \n", err)
		return err, nil, nil, nil, nil, nil
	}

	configData := &struct {
		UsedPublicConfig UsedPublicConfig `yaml:"used_public_config" json:"used_public_config"`
		MysqlMylConfig   MysqlMyConfig    `yaml:"mysql_config" json:"mysql_config"`
	}{}
	//处理config
	transformMyConfig := func(datamap string, configData interface{}, myConfigData interface{}) error {
		confBy, err := yaml2.YAMLToJSON([]byte(datamap))
		if err != nil {
			fmt.Printf(" LoadConfigMap yaml.YAMLToJSON: %v \n", err)
			return err
		}
		err = json.Unmarshal(confBy, configData)
		if err != nil {
			fmt.Printf(" LoadConfigMap yaml.Unmarshal: %v \n", err)
			return err
		}
		res := gjson.Get(string(confBy), MY_CONFIG)
		//myConf = []byte(res.String())
		errB := json.Unmarshal([]byte(res.String()), myConfigData)
		if errB != nil {
			fmt.Printf("LoadConfigMap>>Marshal myConfigData is error,err:%s \n", errB)
			return err
		}
		return nil
	}
	if err = transformMyConfig(datamap, configData, myConfigData); err != nil {
		fmt.Printf("LoadConfigMap>> getMyConfig is error,err:%s \n", err)
		return err, nil, nil, nil, nil, nil
	}
	myConfig = myConfigData
	//获取公共配置
	publicConfigMap := configData.UsedPublicConfig.GetPublicConfigMap()
	if publicConfigMap != nil {

		if publicConfigMap, err = initCommentConfigMap(ctx, namespace, PUBILC_CONFIG_NAME, publicConfigMap); err != nil {

			fmt.Printf(" configmap InitCommentConfigMap: %v \n", err)
			return err, nil, nil, nil, nil, nil
		} else {
			for _, cVaule := range publicConfigMap {
				switch vaule := cVaule.(type) {
				case *KafkaPublicConfig:
					kafkaConfig = vaule
				case *MongoDBPublicConfig:
					mongoDBConfig = vaule
				case *MysqlPublicConfig:
					if len(vaule.Url) > 0 {
						vaule.Url = strings.Replace(vaule.Url, "{username}", configData.MysqlMylConfig.Username, -1)
						vaule.Url = strings.Replace(vaule.Url, "{password}", configData.MysqlMylConfig.Password, -1)
						vaule.Url = strings.Replace(vaule.Url, "{db_name}", configData.MysqlMylConfig.DBName, -1)
					}
					mysqlConfig = vaule
				case *RedisPublicConfig:
					redisConfig = vaule
				}
			}
		}
	}
	//监听配置变化，若变化 ，发布订阅事件
	watchCh := make(chan map[string]interface{}, len(AllPublicConfigName)+3)
	go func() {
		w, err := k8sManager.WatchConfigMap(ctx, namespace, PUBILC_CONFIG_NAME)
		if err != nil {
			fmt.Printf(" configmap Watch: %v \n", err)
			return
		}
		for {
			select {
			case e, _ := <-w.ResultChan():
				obj := e.Object
				// 转成config对象
				cf := obj.(*v1.ConfigMap)
				fmt.Printf("*********************** %s %s ***********************\n", e.Type, cf.Name)
				if e.Type == watch.Added || e.Type == watch.Modified {
					for _, conName := range AllPublicConfigName {
						if _, ok := publicConfigMap[conName]; !ok {
							continue
						}
						//如果堆积过多未消费，则剔除出最先进入得数据
						if len(watchCh) > len(AllPublicConfigName) {
							discard := <-watchCh
							for k, v := range discard {
								fmt.Printf("WatchConfigMap:configName %s  is discard,content:%s \n", k, v)
							}
						}
						switch conName {
						case Kafka_Con:
							kafkaConfig = new(KafkaPublicConfig)
							decoder := yaml.NewDecoder(strings.NewReader(cf.Data[conName]))
							if err = decoder.Decode(kafkaConfig); err != nil {
								fmt.Printf(" Failed to get initCommentConfigMap configType%s, err: %v \n", cf.Data[conName], err)
							} else {
								conMap := make(map[string]interface{})
								conMap[conName] = kafkaConfig
								watchCh <- conMap
								//ConfigEventBus.Publish(Kafka_Con, kafkaPublicConfig)
							}
						case Mong_Con:
							mongoDBConfig = new(MongoDBPublicConfig)
							decoder := yaml.NewDecoder(strings.NewReader(cf.Data[conName]))
							if err = decoder.Decode(mongoDBConfig); err != nil {
								fmt.Printf(" Failed to get initCommentConfigMap configType%s, err: %v \n", cf.Data[conName], err)
							} else {
								conMap := make(map[string]interface{})
								conMap[conName] = mongoDBConfig
								watchCh <- conMap
							}
						case Mysql_Con:
							mysqlConfig = new(MysqlPublicConfig)
							decoder := yaml.NewDecoder(strings.NewReader(cf.Data[conName]))
							if err = decoder.Decode(mysqlConfig); err != nil {
								fmt.Printf(" Failed to get initCommentConfigMap configType%s, err: %v \n", cf.Data[conName], err)
							} else {
								if len(mysqlConfig.Url) > 0 {
									mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{username}", configData.MysqlMylConfig.Username, -1)
									mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{password}", configData.MysqlMylConfig.Password, -1)
									mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{db_name}", configData.MysqlMylConfig.DBName, -1)
								}
								conMap := make(map[string]interface{})
								conMap[conName] = mysqlConfig
								watchCh <- conMap
							}
						case Redis_Con:
							redisConfig = new(RedisPublicConfig)
							decoder := yaml.NewDecoder(strings.NewReader(cf.Data[conName]))
							if err = decoder.Decode(redisConfig); err != nil {
								fmt.Printf(" Failed to get initCommentConfigMap configType%s, err: %v \n", cf.Data[conName], err)
							} else {
								conMap := make(map[string]interface{})
								conMap[conName] = redisConfig
								watchCh <- conMap
							}
						}
					}

				}
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()
	go func() {
		w, err := k8sManager.WatchConfigMap(ctx, namespace, serviceName)
		if err != nil {
			fmt.Printf(" configmap Watch: %v \n", err)
			return
		}
		for {
			select {
			case e := <-w.ResultChan():
				obj := e.Object
				// 转成config对象
				cf := obj.(*v1.ConfigMap)
				fmt.Printf("*********************** %s %s ***********************\n", e.Type, cf.Name)
				if (e.Type == watch.Added || e.Type == watch.Modified) && cf.Name == serviceName {
					if err = transformMyConfig(cf.Data[ymlName], configData, myConfigData); err != nil {
						fmt.Printf("LoadConfigMap>> getMyConfig is error,err:%s \n", err)
						continue
					}
					conMap := make(map[string]interface{})
					conMap[MY_CONFIG] = myConfigData
					//如果堆积过多未消费，则剔除出最先进入得数据
					if len(watchCh) > len(AllPublicConfigName) {
						discard := <-watchCh
						for k, v := range discard {
							fmt.Printf("WatchConfigMap:configName %s  is discard,content:%s \n", k, v)
						}
					}
					watchCh <- conMap
					//关联更新
					if mysqlConfig != nil {
						if len(mysqlConfig.Url) > 0 {
							mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{username}", configData.MysqlMylConfig.Username, -1)
							mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{password}", configData.MysqlMylConfig.Password, -1)
							mysqlConfig.Url = strings.Replace(mysqlConfig.Url, "{db_name}", configData.MysqlMylConfig.DBName, -1)
						}
						conMysqlMap := make(map[string]interface{})
						conMysqlMap[Mysql_Con] = mysqlConfig
						watchCh <- conMysqlMap
					}
				}
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()
	go func() {
		for {
			conMap := <-watchCh
			for k, v := range conMap {
				ConfigEventBus.Publish(k, v)
			}
		}
	}()
	err = nil
	return
}
