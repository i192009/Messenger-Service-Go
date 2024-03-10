package xutil

import (
	"os"
	"strings"
)

// ParsePath parse path
func ParsePath(path string) (realpath string, err error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		realpath = home + path[1:]
	} else {
		realpath = path
	}

	return
}
