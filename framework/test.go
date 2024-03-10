package framework

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"

	"gitlab.zixel.cn/go/framework/variant"
)

type TestRequest struct {
	Method  string            // GET, POST, PUT, DELETE
	Path    string            // URL
	Body    []byte            // Body
	Headers map[string]string // Headers
}

func DoTestRequest(r TestRequest) (obj variant.AbstractValue, code int, err error) {
	if len(r.Body) == 0 {
		r.Body = []byte{}
	}

	req := httptest.NewRequest(r.Method, r.Path, bytes.NewReader(r.Body))

	if len(r.Headers) > 0 {
		for k, v := range r.Headers {
			req.Header.Set(k, v)
		}
	}

	w := httptest.NewRecorder()
	webServer.GetEngine().ServeHTTP(w, req)

	var res any
	err = json.Unmarshal(w.Body.Bytes(), &res)
	if err != nil {
		return nil, w.Code, err
	}

	return variant.New(res), w.Code, err
}
