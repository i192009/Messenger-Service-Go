package xutil

import "reflect"

// get validator rules from struct type
func ExtractTagReflect(tagname string, typ reflect.Type) map[string]any {
	if typ.Kind() != reflect.Struct {
		return nil
	}

	m := make(map[string]any)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		name := f.Name
		if f.Type.Kind() == reflect.Struct {
			m[name] = ExtractTagReflect(tagname, f.Type)
		} else {
			if rule, ok := f.Tag.Lookup(tagname); ok {
				m[name] = rule
			}
		}
	}

	return m
}

// get validator rules from struct
func ExtractTag(tagname string, d any) map[string]any {
	return ExtractTagReflect(tagname, reflect.TypeOf(d))
}
