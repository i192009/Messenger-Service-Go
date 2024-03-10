package xutil

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
)

func FileHashName(params interface{}, split []int, prefix string) string {
	bin, err := json.Marshal(params)
	if err != nil {
		return ""
	}

	dig := md5.Sum(bin)
	key := hex.EncodeToString(dig[:])

	keys := make([]string, 0, len(split))
	start := 0
	for _, length := range split {
		if start+length > len(key) {
			break
		}

		keys = append(keys, key[start:start+length])
		start = start + length
	}

	if start < len(key) {
		keys = append(keys, key[start:])
	}

	path := filepath.Join(prefix, filepath.Join(keys...))

	return path
}
