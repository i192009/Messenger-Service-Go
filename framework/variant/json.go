package variant

import (
	"encoding/json"
	"fmt"
	"os"

	"gitlab.zixel.cn/go/framework/xutil"
)

func LoadJson(data []byte) (ret AbstractValue, err error) {
	var v Variant
	if err = json.Unmarshal(data, &v.ref); err != nil {
		if e, ok := err.(*json.SyntaxError); ok {
			var offset int64
			var line = 0
			for offset = e.Offset; offset > 0; offset-- {
				if data[offset] == '\n' {
					line++
					if line == 5 {
						break
					}
				}
			}

			var info string
			if offset < int64(len(data)) {
				if e.Offset >= int64(len(data)) {
					e.Offset = int64(len(data))
				}
				info = string(data[offset:e.Offset])
			}

			fmt.Print(e.Error(), "offset = ", e.Offset, info)
		}

		return
	}

	ret = &v
	return
}

func SaveJson(val AbstractValue) (data []byte, err error) {
	data, err = json.Marshal(val.Raw())
	return
}

func LoadJsonFile(configEncrypted bool, filename string, fixedPrivateKey string) (ret AbstractValue, err error) {
	if buf, err := os.ReadFile(filename); err != nil {
		return &Nil, err
	} else {
		if configEncrypted {
			buf = xutil.DecryptDataWithFixedPrivateKey(buf, fixedPrivateKey)
		}
		return LoadJson(buf)
	}
}

func SaveJsonFile(val AbstractValue, filename string) error {
	if buf, err := json.Marshal(val.Raw()); err != nil {
		return err
	} else {
		if err := os.WriteFile(filename, buf, 0644); err != nil {
			return err
		}
	}

	return nil
}
