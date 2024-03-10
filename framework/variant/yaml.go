package variant

import (
	"encoding/json"
	"fmt"
	"os"

	"gitlab.zixel.cn/go/framework/xutil"
	"gopkg.in/yaml.v3"
)

func LoadYaml(data []byte) (ret AbstractValue, err error) {
	var v Variant
	datastr := os.ExpandEnv(string(data))
	if err := yaml.Unmarshal([]byte(datastr), &v.ref); err != nil {
		if e, ok := err.(*yaml.TypeError); ok {
			fmt.Print(e.Error())
		}
	}

	ret = &v
	return
}

func SaveYaml(val AbstractValue) (data []byte, err error) {
	data, err = yaml.Marshal(val.Raw())
	return
}

func LoadYamlFile(configEncrypted bool, filename string, fixedPrivateKey string) (ret AbstractValue, err error) {
	if buf, err := os.ReadFile(filename); err != nil {
		return &Nil, err
	} else {
		if configEncrypted {
			buf = xutil.DecryptDataWithFixedPrivateKey(buf, fixedPrivateKey)
		}
		return LoadYaml(buf)
	}
}

func SaveYamlFile(val AbstractValue, filename string) error {
	if buf, err := json.Marshal(val.Raw()); err != nil {
		return err
	} else {
		if err := os.WriteFile(filename, buf, 0644); err != nil {
			return err
		}
	}

	return nil
}
