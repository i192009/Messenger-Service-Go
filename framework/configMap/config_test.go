package configMap

import (
	"testing"
)

type ConfigDemo struct {
	RedisConfig *RedisPublicConfig `yaml:"redis_config"`
	KafkaUrl    string             `yaml:"kafka_url"`
	MysqlConfig *MysqlPublicConfig `yaml:"mysql_config"`
	MyConfig    *MyDemoConfig      `yaml:"my_config"`
}

type MyDemoConfig struct {
	LogLevel string `yaml:"log_level" json:"log_level"`
	LogPath  string `yaml:"log_path" json:"log_path"`
}

var ConfDemo ConfigDemo

func Test_Config(t *testing.T) {
	//myConfig := new(MyDemoConfig)
	//filePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	//fs, err := os.Open(filePath)
	//if err != nil {
	//	t.Fatalf("Open file error, err = %v", err)
	//	return
	//}
	//
	//defer fs.Close()
	//var namespaceBy []byte
	//namespaceBy, err = io.ReadAll(fs)
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//namespaceBy, err := os.ReadFile(filePath)
	//err, kafkaConfig, mysqlConfig, redisConfig, _, myConfC := LoadConfigMap(context.TODO(), string(namespaceBy), "config.yml", "klog-conf", myConfig)
	//if err != nil {
	//	t.Fatalf("LoadConfigMap is error,err:%v \n", err)
	//	return
	//}
	//ConfDemo.MysqlConfig = mysqlConfig
	//ConfDemo.KafkaUrl = kafkaConfig.Url
	//ConfDemo.RedisConfig = redisConfig
	//myConf := <-myConfC
	//errB := json.Unmarshal(myConf, myConfig)
	//if errB != nil {
	//	t.Fatalf("LoadConfigMap>>Marshal is error,err:%v \n", errB)
	//}
	//ConfDemo.MyConfig = myConfig
	////监听配置变化
	//go func() {
	//	for {
	//		myConf := <-myConfC
	//		errB := json.Unmarshal(myConf, myConfig)
	//		if errB != nil {
	//			fmt.Printf("LoadConfigMap>>Marshal is error,err:%v \n", errB)
	//		}
	//		ConfDemo.MyConfig = myConfig
	//	}
	//}()
	//myConfig = myConf.(*MyDemoConfig)
	//ConfDemo.MyConfig = myConfig
	////监听配置变化
	//ConfigEventBus.Subscribe(MY_CONFIG, func(myConfig *MyDemoConfig) {
	//	fmt.Printf("load myConfig,data:%s \n", myConfig)
	//	ConfDemo.MyConfig = myConfig
	//})
	//ConfigEventBus.Subscribe(Mysql_Con, func(myConfig *MysqlPublicConfig) {
	//	fmt.Printf("load mysql,data:%s \n", myConfig)
	//})
	//return
}
