package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"gitlab.zixel.cn/go/framework/logger"
	"gitlab.zixel.cn/go/framework/variant"
	"gitlab.zixel.cn/go/framework/xutil"
)

var o variant.AbstractValue
var lock sync.RWMutex
var log = logger.Get()

var K8S_Namespace string
var K8S_ConfigMap string
var K8S_ConfigKey string

func init() {
	cwd := os.Getenv("PWD")
	log.Infoln("current working directory: ", cwd)
	_ = godotenv.Load(".env", "conf/.env")

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "."
	}

	K8S_Namespace = os.Getenv("PAAS_NAMESPACE")

	if K8S_Namespace != "" {
		log.Println("read from configmap")
		configEncrypted := false
		// 固定读取配置license的configMap
		if err := Watch(false, K8S_Namespace, xutil.LICENSE_CONFIGMAP, xutil.LICENSE_CONFIGKEY); err != nil {
			log.Printf("watch license configmap error: %s\n", err.Error())
		} else if GetString(xutil.LICENSE_SERVER_URL, "") != "" {
			// 如果读取成功则说明需要解密configMap
			configEncrypted = true
			// 启动时证书授权校验
			validateLicenseWhenStart()
			// 注册证书授权校验定时任务
			validateLicenseScheduled()
		}

		// if o != nil {
		// 	// 此时可能没有本地配置文件
		// 	variant.Print(o, "  ", "")
		// }

		K8S_ConfigMap = os.Getenv("CONFIG_MAP")
		K8S_ConfigKey = os.Getenv("CONFIG_KEY")
		K8S_ConfigKeys := strings.Split(K8S_ConfigKey, ",")
		log.Printf("watch configmap '%s/%s, key=%s'...\n", K8S_Namespace, K8S_ConfigMap, K8S_ConfigKey)
		if err := Watch(configEncrypted, K8S_Namespace, K8S_ConfigMap, K8S_ConfigKeys...); err != nil {
			log.Printf("watch configmap error: %s\n", err.Error())
		}
	} else {
		log.Print("start loading path '", configPath, "'...\n")
		_ = Load(false, configPath)
	}

	initConfigVariants()
}

func Load(configEncrypted bool, filenames ...string) error {
	for _, filename := range filenames {
		if fi, err := os.Stat(filename); err != nil {
			return err
		} else if fi.IsDir() {
			err = filepath.Walk(filename, func(fname string, finfo os.FileInfo, err error) error {
				extName := filepath.Ext(fname)
				if !finfo.IsDir() {
					log.Print("start loading file '", fname, "'...\n")
					switch extName {
					case ".conf", ".json":
						v, err := variant.LoadJsonFile(configEncrypted, fname, GetString(xutil.LICENSE_FIXED_PRIVATE_KEY, ""))
						if err != nil {
							return err
						}

						o = variant.Merge(v, o)
					case ".yaml", ".yml":
						v, err := variant.LoadYamlFile(configEncrypted, fname, GetString(xutil.LICENSE_FIXED_PRIVATE_KEY, ""))
						if err != nil {
							return err
						}

						o = variant.Merge(v, o)
					}
				}
				return nil
			})

			if err != nil {
				return err
			}
		} else if v, err := variant.LoadJsonFile(configEncrypted, filename, GetString(xutil.LICENSE_FIXED_PRIVATE_KEY, "")); err != nil {
			return err
		} else {
			o = variant.Merge(v, o)
		}
	}

	return nil
}

func Exists(key string) bool {
	lock.RLock()
	defer lock.RUnlock()
	return !variant.Get(o, key).IsNil()
}

func GetBoolean(key string, defaultValue bool) bool {
	lock.RLock()
	defer lock.RUnlock()
	v := variant.Get(o, key)
	if v.IsBoolean() {
		return v.ToBoolean()
	}

	return defaultValue
}

func GetInt(key string, defaultValue int64) int64 {
	lock.RLock()
	defer lock.RUnlock()
	v := variant.Get(o, key)
	if v.IsInt() {
		return v.ToInt()
	}

	return defaultValue
}

func GetReal(key string, defaultValue float64) float64 {
	lock.RLock()
	defer lock.RUnlock()
	v := variant.Get(o, key)
	if v.IsDecimal() {
		return v.ToDecimal()
	}

	return defaultValue
}

func GetString(key string, defaultValue string) string {
	lock.RLock()
	defer lock.RUnlock()
	v := variant.Get(o, key)
	if !v.IsNil() {
		return v.ToString()
	}

	return defaultValue
}

func GetArray(key string) []any {
	lock.RLock()
	defer lock.RUnlock()
	v := variant.Get(o, key)
	if v.IsArray() {
		return v.ToArray()
	}

	return nil
}

func GetObject(key string) map[string]any {
	lock.RLock()
	defer lock.RUnlock()
	v := variant.Get(o, key)
	if v.IsObject() {
		return v.ToObject()
	}

	return nil
}
