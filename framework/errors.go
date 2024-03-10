package framework

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"gitlab.zixel.cn/go/framework/xutil"
)

type ServiceResponse_t struct {
	Name string `json:"name,omitempty"`
	UUID string `json:"uuid,omitempty"`
}

type ServiceError struct {
	error
	Service    *ServiceResponse_t `json:"service,omitempty"`
	Code       int                `json:"code,omitempty"`
	Message    string             `json:"message,omitempty"`
	Args       []any              `json:"args,omitempty"`
	Stacktrace string             `json:"-"`
}

type ErrorResponse_t struct {
	Error ServiceError `json:"error,omitempty"`
}

var DefaultMessage = ""
var serviceInfo ServiceResponse_t
var errorTips = make(map[int]string)

func SetErrorTips(errcode int, message string) {
	errorTips[errcode] = message
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // 先调用c.Next()执行后面的中间件
		// 所有中间件及router处理完毕后从这里开始执行
		// 检查c.Errors中是否有错误
		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors[0].Err
		if err == nil {
			return
		}

		// 若是自定义的错误则将code、msg返回
		if srv_error, ok := err.(ServiceError); ok {
			srv_error.Service = &serviceInfo
			if srv_error.Message == DefaultMessage {
				if srv_error.Message, ok = errorTips[srv_error.Code]; ok && len(srv_error.Args) > 0 {
					srv_error.Message = err.Error()
				}
			}
			c.JSON(http.StatusOK, ErrorResponse_t{
				Error: srv_error,
			})
			return
		}

		// 若非自定义错误则返回详细错误信息err.Error()
		// 比如save session出错时设置的err
		c.JSON(http.StatusOK, ErrorResponse_t{
			Error: ServiceError{
				Service: &serviceInfo,
				Code:    500,
				Message: "服务器异常",
			},
		})
	}
}

func SetServiceDetail(name string, uuid string) {
	serviceInfo.Name = name
	serviceInfo.UUID = uuid
}

func NewError(c *gin.Context, err ErrorCode, args ...interface{}) {
	Errorf(c, err.Code, err.Message, args)
}

/**
 * 如果抛出的err是ServiceError则直接使用其code，防止code被吞掉
 * @param c 请求上下文
 * @param err 捕获的err
 * @param defaultCode 默认错误码
 */
func ErrorfWithCode(c *gin.Context, err error, defaultCode int) {
	if serr, ok := err.(ServiceError); ok {
		Errorf(c, serr.Code, serr.Error())
	} else {
		Errorf(c, defaultCode, err.Error())
	}
}

func Errorf(c *gin.Context, code int, message string, args ...interface{}) {
	err := NewServiceError(code, message, args...)
	log.Errorf("%v", err.Stacktrace)
	c.Abort()
	c.Error(err)
}

func Error(c *gin.Context, err error) {
	if se, ok := err.(ServiceError); ok {
		log.Errorf("%v", se.Stacktrace)
	}
	c.Abort()
	c.Error(err)
}

func NewServiceError(code int, message string, args ...any) ServiceError {
	if message == "" {
		if defaultMessage, ok := errorTips[code]; ok {
			message = defaultMessage
		}
	}

	for i := 0; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		if strings.HasSuffix(file, filepath.Join("framework", "server", "errors.go")) {
			continue
		}
		if strings.HasSuffix(file, filepath.Join("framework", "errors.go")) {
			continue
		}
		fname := filepath.Base(file)
		log.Errorf("#[%s:%d] 0x%x error code = %d, message = %s", fname, line, pc, code, message)
		break
	}

	err := ServiceError{
		error:   fmt.Errorf(message, args...),
		Code:    code,
		Message: message,
		Args:    args,
	}
	err.Stacktrace = xutil.StackTrace(err, "---")
	return err
}
