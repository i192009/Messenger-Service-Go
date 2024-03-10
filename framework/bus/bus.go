package bus

import (
	"fmt"
	"reflect"
	"sync"

	"gitlab.zixel.cn/go/framework/logger"
	"gitlab.zixel.cn/go/framework/xutil"
)

var log = logger.Get()

// EventBus EventBus
type EventBus interface {
	Subscribe(topic string, handler interface{}) error
	Publish(topic string, args ...interface{})
}

// AsyncEventBus 异步事件总线
type AsyncEventBus struct {
	handlers map[string][]reflect.Value
	lock     sync.Mutex
}

// NewAsyncEventBus new
func NewAsyncEventBus() *AsyncEventBus {
	return &AsyncEventBus{
		handlers: map[string][]reflect.Value{},
		lock:     sync.Mutex{},
	}
}

// Subscribe 订阅
func (bus *AsyncEventBus) Subscribe(topic string, f interface{}) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()

	v := reflect.ValueOf(f)
	if v.Type().Kind() != reflect.Func {
		return fmt.Errorf("handler is not a function")
	}

	handler, ok := bus.handlers[topic]
	if !ok {
		handler = []reflect.Value{}
	}
	handler = append(handler, v)
	bus.handlers[topic] = handler

	return nil
}

// Unsubscribe 退订
func (bus *AsyncEventBus) Unsubscribe(topic string, f interface{}) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()

	v := reflect.ValueOf(f)
	if v.Type().Kind() != reflect.Func {
		return fmt.Errorf("handler is not a function")
	}

	handler, ok := bus.handlers[topic]
	if !ok {
		handler = []reflect.Value{}
	}

	for i, h := range handler {
		if h == v {
			handler = append(handler[:i], handler[i+1:]...)
			break
		}
	}

	bus.handlers[topic] = handler
	return nil
}

// Publish 发布
// 这里异步执行，并且不会等待返回结果
func (bus *AsyncEventBus) Publish(topic string, args ...interface{}) {
	handlers, ok := bus.handlers[topic]
	if !ok {
		log.Warn("not found handlers in topic:", topic)
		return
	}

	params := make([]reflect.Value, len(args))
	for i, arg := range args {
		params[i] = reflect.ValueOf(arg)
	}

	for i := range handlers {
		go func(idx int) {
			log.Debug("call ", topic, " args = ", args)

			defer func() {
				if err := recover(); err != nil {
					log.Debugf("call %s error, stack frames :%s", topic, xutil.StackTrace(err, "    "))
				}
			}()

			handlers[idx].Call(params)
		}(i)
	}
}
