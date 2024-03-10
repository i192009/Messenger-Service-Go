package k8smanager_test

import (
	"os"
	"testing"
	"time"

	"gitlab.zixel.cn/go/framework/k8smanager"
)

func TestListenConfigMap(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_CONF", "~/.kube/config")
	k8sClientManager := k8smanager.NewK8sClientManager()
	c, err := k8sClientManager.ListenConfigMap("default", "partner-conf")
	if err != nil {
		t.Error(err)
		return
	}

	var abort bool = false
	for !abort {
		select {
		case v := <-c:
			t.Log(v)
		case <-time.After(10 * time.Second):
			t.Log("timeout")
			abort = true
		}
	}

	k8sClientManager.StopListenConfigMap("default", "partner-conf")
}
