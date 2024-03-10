package xutil

import "reflect"

var indexTypes = []reflect.Kind{
	reflect.Int,
	reflect.Int8,
	reflect.Int16,
	reflect.Int32,
	reflect.Int64,
	reflect.Uint,
	reflect.Uint8,
	reflect.Uint16,
	reflect.Uint32,
	reflect.Uint64,
}

// 判断是否可以作为数组下标
func IsIndexType(t reflect.Type) bool {
	return InSlice(t.Kind(), indexTypes)
}

// 判断是否由某个类派生
func IsKindOf[D any](o any) bool {
	return IsKindOfType(reflect.TypeOf(o), reflect.TypeOf((*D)(nil)).Elem())
}

// 判断是否由某个类派生
func IsKindOfType(sourceType reflect.Type, targetType reflect.Type) bool {
	switch k := sourceType.Kind(); k {
	case reflect.Bool:
		return k == targetType.Kind()
	case reflect.Int:
		return k == targetType.Kind()
	case reflect.Int8:
		return k == targetType.Kind()
	case reflect.Int16:
		return k == targetType.Kind()
	case reflect.Int32:
		return k == targetType.Kind()
	case reflect.Int64:
		return k == targetType.Kind()
	case reflect.Uint:
		return k == targetType.Kind()
	case reflect.Uint8:
		return k == targetType.Kind()
	case reflect.Uint16:
		return k == targetType.Kind()
	case reflect.Uint32:
		return k == targetType.Kind()
	case reflect.Uint64:
		return k == targetType.Kind()
	case reflect.Uintptr:
		return k == targetType.Kind()
	case reflect.Float32:
		return k == targetType.Kind()
	case reflect.Float64:
		return k == targetType.Kind()
	case reflect.Complex64:
		return k == targetType.Kind()
	case reflect.Complex128:
		return k == targetType.Kind()
	case reflect.Array:
		if k == targetType.Kind() {
			return IsKindOfType(sourceType.Elem(), targetType.Elem())
		}
	case reflect.Chan:
	case reflect.Func:
	case reflect.Interface:
	case reflect.Map:
		if k == targetType.Kind() {
			return IsKindOfType(sourceType.Key(), targetType.Key()) && IsKindOfType(sourceType.Elem(), targetType.Elem())
		}
	case reflect.Pointer:
		if k == targetType.Kind() {
			return IsKindOfType(sourceType.Elem(), targetType.Elem())
		}
	case reflect.Slice:
		if k == targetType.Kind() {
			return IsKindOfType(sourceType.Elem(), targetType.Elem())
		}
	case reflect.String:
		return k == targetType.Kind()
	case reflect.Struct:
		for i := 0; i < sourceType.NumField(); i++ {
			field := sourceType.Field(i)
			if field.Name == field.Type.Name() && IsKindOfType(sourceType, field.Type) {
				return true
			}
		}

		return false
	case reflect.UnsafePointer:
		return false
	}

	return false
}
