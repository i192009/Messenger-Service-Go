package xutil_test

import (
	"testing"

	"gitlab.zixel.cn/go/framework/xutil"
	"go.mongodb.org/mongo-driver/bson"
)

type A struct {
}

type B struct {
	A
}

func IsKindOf_Test(t *testing.T) {
	b := B{}
	if ok := xutil.IsKindOf[A](b); !ok {
		t.Fatalf("error kind of")
	}

	c := map[string]any{}
	if ok := xutil.IsKindOf[bson.M](c); !ok {
		t.Fatalf("error kind of")
	}
}
