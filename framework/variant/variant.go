package variant

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type AbstractVisitor func(any, AbstractValue, AbstractValue) // key, value, parent

type AbstractValue interface {
	Type() reflect.Type
	Raw() any
	Len() int
	Exists(any) bool
	Get(any) AbstractValue
	Set(any, any) bool
	Traveral(AbstractVisitor)
	GetBoolean(key any, defaultValue bool) bool
	GetInt(key any, defaultValue int64) int64
	GetReal(key any, defaultValue float64) float64
	GetStr(key any, defaultValue string) string
	GetArray(key any, defaultValue []any) []any
	GetObject(key any, defaultValue map[string]any) map[string]any

	IsNil() bool
	IsNumeric() bool
	IsInteger() bool
	IsDecimal() bool
	IsArray() bool
	IsObject() bool

	IsInt() bool
	IsUint() bool
	IsString() bool
	IsBoolean() bool
	IsTime() bool

	ToInt() int64
	ToUint() uint64
	ToDecimal() float64
	ToString() string
	ToBoolean() bool
	ToTime() time.Time
	ToArray() []any
	ToObject() map[string]any
	ToInterface() any
}

func get(j AbstractValue, keys []string) AbstractValue {
	if j == nil {
		return &Nil
	}
	if len(keys) == 0 {
		return &Nil
	}

	if j.IsObject() || j.IsArray() {
		if len(keys) == 1 {
			return j.Get(keys[0])
		}

		return get(j.Get(keys[0]), keys[1:])
	}

	return &Nil
}

func set(j AbstractValue, keys []string, value any) bool {
	if len(keys) == 0 {
		return false
	}

	if j.IsObject() || j.IsArray() {
		v := j.Get(keys[0])
		if set(v, keys[1:], value) {
			j.Set(keys[0], v)
		}

		return true
	}

	return false
}

func Get(j AbstractValue, keys string) AbstractValue {
	return get(j, strings.Split(keys, "."))
}

func Set(j AbstractValue, keys string, value any) bool {
	return set(j, strings.Split(keys, "."), value)
}

func Copy(j AbstractValue) AbstractValue {
	var ret AbstractValue
	if j.IsObject() {
		ret = New(map[string]any{})
		for k, v := range j.ToObject() {
			ret.Set(k, Copy(New(v)))
		}
		return ret
	} else if j.IsArray() {
		ret = New(make([]any, j.Len()))
		for k, v := range j.ToArray() {
			ret.Set(k, Copy(New(v)))
		}
		return ret
	} else {
		return New(j)
	}
}

func Print(j AbstractValue, indent string, indents string) {
	var traveral AbstractVisitor
	traveral = func(key any, value AbstractValue, parent AbstractValue) {
		prefix := ""
		if parent.IsObject() {
			prefix = fmt.Sprintf("%s%v: ", indents, key)
		} else if parent.IsArray() {
			prefix = indents
		}

		if value.IsObject() {
			fmt.Printf("%s{\n", prefix)
			indents = indents + indent
			value.Traveral(traveral)
			indents = indents[:len(indents)-len(indent)]
			fmt.Printf("%s},\n", indents)
		} else if value.IsArray() {
			fmt.Printf("%s[\n", prefix)
			indents = indents + indent
			value.Traveral(traveral)
			indents = indents[:len(indents)-len(indent)]
			fmt.Printf("%s],\n", indents)
		} else {
			if value.IsString() {
				fmt.Printf("%s\"%s\",\n", prefix, value.ToString())
			} else {
				fmt.Printf("%s %s,\n", prefix, value.ToString())
			}
		}
	}

	j.Traveral(traveral)
}

// 合并newValue、oldValue两个配置，key相同时newValue会覆盖oldValue的值
func Merge(newValue, oldValue AbstractValue, mn ...AbstractValue) AbstractValue {
	merge := func(m1, m2 AbstractValue) AbstractValue {
		if m1 == nil {
			return nil
		}

		if m2 == nil {
			return nil
		}

		if m1.Type() != m2.Type() {
			return nil
		}

		if !m2.IsObject() && !m2.IsArray() {
			return nil
		}

		// 拷贝m2，避免操作原数据
		m2copy := Copy(m2)

		var stack = []AbstractValue{m2copy}
		var traveral AbstractVisitor
		traveral = func(key any, value AbstractValue, parent AbstractValue) {
			// 数组最后一个元素为当前操作值对象
			currentValue := stack[len(stack)-1]
			if value.IsObject() {
				dst := currentValue.Get(key)
				if dst.IsNil() {
					// m2中为空则直接添加m1的值
					currentValue.Set(key, value)
				} else if !dst.IsObject() {
					// m2中不为空且不是对象也直接添加m1的值
					currentValue.Set(key, value)
				} else {
					// m2中不为空且是对象则进行递归查找
					stack = append(stack, dst) // 当前值对象追加到数组末尾，对应函数开头取操作对象的逻辑
					value.Traveral(traveral)
					stack = stack[:len(stack)-1]
					// 设置递归后的值
					currentValue.Set(key, dst)
				}
			} else if value.IsArray() {
				dst := currentValue.Get(key)
				if dst.IsNil() {
					// m2中为空则直接添加m1的值
					currentValue.Set(key, value)
				} else if !dst.IsArray() {
					// m2中不为空且不是数组也直接添加m1的值
					currentValue.Set(key, value)
				} else {
					// m2中不为空且是数组则进行数组合并
					array2 := dst.ToArray()
					currentValue.Set(key, array2)
				}
			} else {
				currentValue.Set(key, value)
			}
		}

		m1.Traveral(traveral)

		return m2copy
	}

	var m AbstractValue
	if newValue == nil {
		m = oldValue
	} else if oldValue == nil {
		m = newValue
	} else {
		m = merge(newValue, oldValue)
	}

	for _, e := range mn {
		if m == nil {
			m = e
		} else if e != nil {
			m = merge(m, e)
		}
	}

	return m
}
