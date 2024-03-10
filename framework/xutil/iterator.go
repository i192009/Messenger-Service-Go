package xutil

import (
	"fmt"
	"reflect"
)

type Iterator interface {
	OnEnter(k any, v any) // 进入对象/迭代数组前调用
	OnLeave(k any, v any) // 离开对象/迭代数组后调用
}

type LookupIterator struct {
	Tab    string
	Intent string
}

func NewLookupIterator(tab int) LookupIterator {
	var space = ""
	for i := 0; i < tab; i++ {
		space = space + "-"
	}

	return LookupIterator{
		Tab:    space,
		Intent: "",
	}
}

func (s *LookupIterator) OnEnter(k any, v any) {
	s.Intent = s.Intent + s.Tab
	if k != nil {
		fmt.Printf("%s%q:", s.Intent, k)
		t := reflect.TypeOf(v)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		switch t.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
			fmt.Print("\n")
		}
	}
}

func (s *LookupIterator) OnLeave(k any, v any) {
	s.Intent = s.Intent[len(s.Tab):]
	fmt.Print("\n")
}

func (s LookupIterator) LookupVal(v any) {
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Pointer:
		r := reflect.ValueOf(v)
		s.LookupVal(r.Elem().Interface())
	case reflect.Struct:
		s.LookupStruct(v)
	case reflect.Map:
		s.LookupMap(v)
	case reflect.Array:
		s.LookupArray(v)
	case reflect.Slice:
		s.LookupSlice(v)
	case reflect.String:
		fmt.Print(v.(string))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Print(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fmt.Print(v)
	case reflect.Float32, reflect.Float64:
		fmt.Print(v)
	case reflect.Bool:
		fmt.Print(v.(bool))
	default:
		fmt.Printf("%q", v)
	}
}

func (s LookupIterator) LookupStruct(v any) {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Struct {
		panic("the value is not a structure")
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := reflect.ValueOf(v).Field(i).Interface()
		s.OnEnter(f.Name, v)
		s.LookupVal(v)
		s.OnLeave(f.Name, v)
	}
}

func (s LookupIterator) LookupArray(v any) {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Array {
		panic("the value is not a array")
	}

	r := reflect.ValueOf(v)
	for i := 0; i < r.Len(); i++ {
		e := r.Index(i).Interface()
		s.OnEnter(i, e)
		s.LookupVal(e)
		s.OnLeave(i, e)
	}
}

func (s LookupIterator) LookupSlice(v any) {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Slice {
		panic("the value is not a slice")
	}

	r := reflect.ValueOf(v)
	for i := 0; i < r.Len(); i++ {
		e := r.Index(i).Interface()
		s.OnEnter(i, e)
		s.LookupVal(e)
		s.OnLeave(i, e)
	}
}

func (s LookupIterator) LookupMap(v any) {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Map {
		panic("the value is not a map")
	}

	r := reflect.ValueOf(v)
	it := r.MapRange()
	for it.Next() {
		mk := it.Key().Interface()
		mv := it.Value().Interface()
		s.OnEnter(mk, mv)
		s.LookupVal(mv)
		s.OnLeave(mk, mv)
	}
}

type LookupIteratorTest1 struct {
	F1 string
	F2 map[string]int
}

type LookupIteratorTest2 struct {
	F1 string
	F2 int
	F3 map[string]string
	F4 map[string]*LookupIteratorTest1
}

func LookupIteratorTest() {
	var test1 LookupIteratorTest1 = LookupIteratorTest1{
		F1: "LookupIteratorTest1-1",
		F2: map[string]int{"f1": 1, "f2": 2, "f3": 3, "f4": 4},
	}

	var test2 LookupIteratorTest1 = LookupIteratorTest1{
		F1: "LookupIteratorTest1-2",
		F2: map[string]int{"f1": 4, "f2": 5, "f3": 6, "f4": 7},
	}

	var test0 LookupIteratorTest2 = LookupIteratorTest2{
		F1: "LookupIteratorTest2",
		F2: 1000,
		F3: map[string]string{
			"aaa": "123",
			"bbb": "4456",
			"ccc": "fjasdf",
		},
		F4: map[string]*LookupIteratorTest1{
			"x1": {
				F1: "LookupIteratorTest1-3",
				F2: map[string]int{"k1": 1, "k2": 2, "k3": 3, "k4": 4},
			},
			"x2": {
				F1: "LookupIteratorTest1-4",
				F2: map[string]int{"j1": 4, "j2": 5, "j3": 6, "j4": 7},
			},
			"o1": &test1,
			"o2": &test2,
		},
	}

	s := NewLookupIterator(4)
	s.LookupVal(test0)
}
