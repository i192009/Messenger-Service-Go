package variant

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gitlab.zixel.cn/go/framework/xutil"
)

type Variant struct {
	ref any
}

var Nil = Variant{ref: nil}

func New(raw any) AbstractValue {
	if raw == nil {
		return &Nil
	}

	switch val := raw.(type) {
	case *Variant:
		return New(*val)
	case Variant:
		if val.IsNil() {
			return &Nil
		}

		return &Variant{ref: val.ref}
	}

	return &Variant{ref: raw}
}

func (j Variant) Type() reflect.Type {
	return reflect.TypeOf(j.ref)
}

func (j Variant) Raw() any {
	return j.ref
}

func (j Variant) Exists(key any) bool {
	return !j.Get(key).IsNil()
}

func (j Variant) Get(key any) AbstractValue {
	rv := reflect.ValueOf(j.ref)
	kv := reflect.ValueOf(key)

	rt := rv.Type()
	kt := kv.Type()
	switch rt.Kind() {

	case reflect.Slice, reflect.Array:
		{
			// 只要是整型类型即可
			if xutil.IsIndexType(kt) {
				idx := int(kv.Int())
				if idx < 0 || idx >= rv.Len() {
					return &Nil
				}
				return New(rv.Index(int(kv.Int())).Interface())
			} else if kt.Kind() == reflect.String {
				kv := reflect.ValueOf(key)
				if idx, err := strconv.ParseInt(kv.String(), 10, 32); err != nil {
					return &Nil
				} else if idx < 0 || idx >= int64(rv.Len()) {
					return &Nil
				} else {
					return New(rv.Index(int(idx)).Interface())
				}
			}
		}
	case reflect.Map:
		{
			if kt != rt.Key() {
				return &Nil
			}

			found := rv.MapIndex(kv)
			if !found.IsValid() {
				return &Nil
			}

			return New(found.Interface())
		}
	}

	return &Nil
}

func (j *Variant) Set(key any, value any) bool {
	rv := reflect.ValueOf(j.ref)
	kv := reflect.ValueOf(key)

	rt := rv.Type()
	kt := kv.Type()

	switch v := value.(type) {
	case Variant:
		value = v.ref
	case *Variant:
		value = v.ref
	}

	switch rt.Kind() {
	case reflect.Pointer:
		return New(rv.Elem().Interface()).Set(key, value)
	case reflect.Slice, reflect.Array:
		{
			if xutil.IsIndexType(kt) {
				idx := int(kv.Int())
				if idx < 0 || idx >= rv.Len() {
					return false
				}

				ele := rv.Index(idx)
				if ele.CanSet() {
					ele.Set(reflect.ValueOf(value))
					return true
				}
			}
		}
	case reflect.Map:
		{
			if kt != rt.Key() {
				return false
			}

			rv.SetMapIndex(kv, reflect.ValueOf(value))
			return true
		}
	}

	return false
}

func (j Variant) Len() int {
	rt := reflect.TypeOf(j.ref)
	if xutil.InSlice(rt.Kind(), []reflect.Kind{reflect.Slice, reflect.Array, reflect.Map, reflect.String}) {
		return reflect.ValueOf(j.ref).Len()
	}

	return -1
}

// 遍历Object对象的Key，Value
func (j Variant) Traveral(visitor AbstractVisitor) {
	rv := reflect.ValueOf(j.ref)
	rt := rv.Type()
	if rt.Kind() == reflect.Map {
		it := rv.MapRange()
		for it.Next() {
			visitor(it.Key().Interface(), New(it.Value().Interface()), &j)
		}
	} else if rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice {
		for i := 0; i < rv.Len(); i++ {
			visitor(i, New(rv.Index(i).Interface()), &j)
		}
	}
}

func (j Variant) GetBoolean(key any, defaultValue bool) bool {
	if val := j.Get(key); !val.IsNil() {
		if val.IsBoolean() {
			return val.ToBoolean()
		}
	}

	return defaultValue
}

func (j Variant) GetInt(key any, defaultValue int64) int64 {
	if val := j.Get(key); !val.IsNil() {
		if val.IsInteger() {
			return val.ToInt()
		}
	}

	return defaultValue
}

func (j Variant) GetReal(key any, defaultValue float64) float64 {
	if val := j.Get(key); !val.IsNil() {
		if val.IsDecimal() {
			return val.ToDecimal()
		}
	}

	return defaultValue
}

func (j Variant) GetStr(key any, defaultValue string) string {
	if val := j.Get(key); !val.IsNil() {
		if val.IsString() {
			return val.ToString()
		}
	}

	return defaultValue
}

func (j Variant) GetArray(key any, defaultValue []any) []any {
	if val := j.Get(key); !val.IsNil() {
		if val.IsArray() {
			return val.ToArray()
		}
	}

	return defaultValue
}

func (j Variant) GetObject(key any, defaultValue map[string]any) map[string]any {
	if val := j.Get(key); !val.IsNil() {
		if val.IsObject() {
			return val.ToObject()
		}
	}

	return defaultValue
}

func (j Variant) ToInt() int64 {
	switch ref := j.ref.(type) {
	case Variant:
		return ref.ToInt()
	case *Variant:
		return ref.ToInt()
	default:
		rv := reflect.ValueOf(j.ref)
		rt := rv.Type()

		tt := reflect.TypeOf(int64(0))
		if rv.CanConvert(tt) {
			return rv.Convert(tt).Int()
		} else if rt.Kind() == reflect.String {
			ret, err := strconv.ParseInt(rv.String(), 10, 64)
			if err == nil {
				return ret
			}
		}
	}

	return 0
}

func (j Variant) ToUint() uint64 {
	switch ref := j.ref.(type) {
	case Variant:
		return ref.ToUint()
	case *Variant:
		return ref.ToUint()
	default:
		rv := reflect.ValueOf(j.ref)
		rt := rv.Type()

		tt := reflect.TypeOf(uint64(0))
		if rv.CanConvert(tt) {
			return rv.Convert(tt).Uint()
		} else if rt.Kind() == reflect.String {
			ret, err := strconv.ParseUint(rv.String(), 10, 64)
			if err == nil {
				return ret
			}
		}
	}

	return 0
}

func (j Variant) ToDecimal() float64 {
	switch ref := j.ref.(type) {
	case Variant:
		return ref.ToDecimal()
	case *Variant:
		return ref.ToDecimal()
	default:
		rv := reflect.ValueOf(j.ref)
		rt := rv.Type()

		tt := reflect.TypeOf(float64(0))
		if rv.CanConvert(tt) {
			return rv.Convert(tt).Float()
		} else if rt.Kind() == reflect.String {
			ret, err := strconv.ParseFloat(rv.String(), 64)
			if err == nil {
				return ret
			}
		}
	}

	return 0
}

func (j Variant) ToString() string {
	switch ref := j.ref.(type) {
	case Variant:
		return ref.ToString()
	case *Variant:
		return ref.ToString()
	default:
		return fmt.Sprint(ref)
	}
}

func (j Variant) ToBoolean() bool {
	switch ref := j.ref.(type) {
	case Variant:
		return ref.ToBoolean()
	case *Variant:
		return ref.ToBoolean()
	default:
		rv := reflect.ValueOf(j.ref)
		rt := rv.Type()

		t1 := reflect.TypeOf(bool(false))
		t2 := reflect.TypeOf(int64(0))
		if rv.CanConvert(t1) {
			return rv.Convert(t1).Bool()
		} else if rv.CanConvert(t2) {
			return rv.Convert(t2).Int() != 0
		} else if rt.Kind() == reflect.String {
			ret, err := strconv.ParseBool(rv.String())
			if err == nil {
				return ret
			}
		}
	}

	return false
}

func (j Variant) ToTime() time.Time {
	switch ref := j.ref.(type) {
	case Variant:
		return ref.ToTime()
	case *Variant:
		return ref.ToTime()
	default:
		rv := reflect.ValueOf(j.ref)
		rt := rv.Type()
		if rt.Kind() == reflect.Struct && rt == reflect.TypeOf(time.Time{}) {
			return rv.Interface().(time.Time)
		} else if rv.CanInt() {
			return time.UnixMilli(rv.Int())
		} else if rv.CanFloat() {
			i, f := math.Modf(rv.Float())
			return time.Unix(int64(i), int64(f*float64(time.Second)))
		} else {
			return time.UnixMilli(0)
		}
	}
}

func (j Variant) ToArray() []any {
	switch ref := j.ref.(type) {
	case Variant:
		return ref.ToArray()
	case *Variant:
		return ref.ToArray()
	default:
		rv := reflect.ValueOf(j.ref)
		rt := rv.Type()
		if rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array || rt.Kind() == reflect.String {
			ret := make([]any, 0, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				ret = append(ret, rv.Index(i).Interface())
			}

			return ret
		}

		if rt.Kind() == reflect.Map {
			keys := rv.MapKeys()
			ret := make([]any, 0, len(keys))
			for i := 0; i < len(keys); i++ {
				ret = append(ret, keys[i].Interface())
			}

			return ret
		}

		return nil
	}
}

func (j Variant) ToObject() map[string]any {
	switch ref := j.ref.(type) {
	case Variant:
		return ref.ToObject()
	case *Variant:
		return ref.ToObject()
	case *map[string]any:
		return *ref
	case map[string]any:
		return ref
	default:
		rv := reflect.ValueOf(j.ref)
		rt := rv.Type()

		kt := rt.Key()

		if rt.Kind() != reflect.Map {
			return nil
		}

		if kt.Kind() == reflect.String && rt.Elem().Kind() == reflect.Interface {
			ret := make(map[string]any)
			keys := rv.MapKeys()
			for _, key := range keys {
				ret[fmt.Sprint(key)] = rv.MapIndex(key).Interface()
			}

			return ret
		}

		return nil
	}
}

func (j Variant) ToInterface() any {
	return j.ref
}

func (j Variant) IsNil() bool {
	switch ref := j.ref.(type) {
	case Variant:
		return ref.IsNil()
	case *Variant:
		return ref.IsNil()
	default:
		return j.ref == nil
	}
}

func (j Variant) IsNumeric() bool {
	return j.IsInt() || j.IsUint() || j.IsDecimal()
}

func (j Variant) IsInteger() bool {
	return j.IsInt() || j.IsUint()
}

func (j Variant) IsDecimal() bool {
	if j.IsNil() {
		return false
	}

	switch ref := j.ref.(type) {
	case Variant:
		return ref.IsNil()
	case *Variant:
		return ref.IsNil()
	default:
		return reflect.ValueOf(j.ref).CanFloat()
	}
}

func (j Variant) IsBoolean() bool {
	if j.IsNil() {
		return false
	}

	switch ref := j.ref.(type) {
	case Variant:
		return ref.IsNil()
	case *Variant:
		return ref.IsNil()
	default:
		rv := reflect.ValueOf(j.ref)
		rt := rv.Type()
		if rt.Kind() == reflect.Bool {
			return true
		} else if rv.CanInt() {
			return rv.Int() == 0 || rv.Int() == 1
		} else if rt.Kind() == reflect.String {
			v := strings.ToLower(rv.String())
			return v == "true" || v == "false" || v == "0" || v == "1"
		} else {
			return false
		}
	}
}

func (j Variant) IsTime() bool {
	if j.IsNil() {
		return false
	}

	switch ref := j.ref.(type) {
	case Variant:
		return ref.IsNil()
	case *Variant:
		return ref.IsNil()
	default:
		rv := reflect.ValueOf(j.ref)
		rt := rv.Type()
		if rt.Kind() == reflect.Struct && rt == reflect.TypeOf(time.Time{}) {
			return true
		} else if rv.CanInt() || rv.CanFloat() {
			return true
		} else {
			return false
		}
	}
}

func (j Variant) IsInt() bool {
	if j.IsNil() {
		return false
	}

	switch ref := j.ref.(type) {
	case Variant:
		return ref.IsNil()
	case *Variant:
		return ref.IsNil()
	default:
		return reflect.ValueOf(j.ref).CanInt()
	}
}

func (j Variant) IsUint() bool {
	if j.IsNil() {
		return false
	}

	switch ref := j.ref.(type) {
	case Variant:
		return ref.IsNil()
	case *Variant:
		return ref.IsNil()
	default:
		return reflect.ValueOf(j.ref).CanUint()
	}
}

func (j Variant) IsString() bool {
	if j.IsNil() {
		return false
	}

	switch ref := j.ref.(type) {
	case Variant:
		return ref.IsNil()
	case *Variant:
		return ref.IsNil()
	default:
		return reflect.TypeOf(j.ref).Kind() == reflect.String
	}
}

func (j Variant) IsArray() bool {
	if j.IsNil() {
		return false
	}

	switch ref := j.ref.(type) {
	case Variant:
		return ref.IsNil()
	case *Variant:
		return ref.IsNil()
	}

	switch reflect.TypeOf(j.ref).Kind() {
	case reflect.Array, reflect.Slice:
		return true
	default:
		return false
	}
}

func (j Variant) IsObject() bool {
	if j.IsNil() {
		return false
	}

	switch ref := j.ref.(type) {
	case Variant:
		return ref.IsObject()
	case *Variant:
		return ref.IsObject()
	default:
		return reflect.TypeOf(j.ref).Kind() == reflect.Map
	}
}
