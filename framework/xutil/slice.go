package xutil

import (
	"reflect"
	"sort"

	"golang.org/x/exp/constraints"
)

// check e in the slice a
func InSlice[E comparable, A []E](e E, a A) bool {
	for _, ele := range a {
		if ele == e {
			return true
		}
	}

	return false
}

// check e in the slice a
func InSliceF[E any, A []E](e E, a A, comp func(lsh E, rsh E) int) bool {
	for _, ele := range a {
		if comp(ele, e) == 0 {
			return true
		}
	}

	return false
}

// reduce slice to any other type
func Reduce[T, M any](s []T, initValue M, f func(M, int, T) M) M {
	acc := initValue
	for i, v := range s {
		acc = f(acc, i, v)
	}
	return acc
}

func Unique[T comparable](a []T) []T {
	r := make([]T, 0, len(a))
	for i, e := range a {
		if !InSlice(e, a[i+1:]) {
			r = append(r, e)
		}
	}

	return r
}

func UniqueF[T any](a []T, comp func(lsh T, rsh T) int) []T {
	r := make([]T, 0, len(a))
	for i, e := range a {
		if !InSliceF(e, a[i+1:], comp) {
			r = append(r, e)
		}
	}

	return r
}

// return the top of the element list, which determin by function fn
func Top[E constraints.Ordered, F func(i, j E) bool](fn F, e1 E, e2 E, etc ...E) E {
	ret := e1

	if fn(e1, e2) {
		ret = e2
	}

	for _, e := range etc {
		if fn(ret, e) {
			ret = e
		}
	}

	return ret
}

// return the min value from the element list
func Min[E constraints.Ordered](i E, j E, etc ...E) E {
	return Top(func(x E, y E) bool { return y < x }, i, j, etc...)
}

// return the max value from the element list
func Max[E constraints.Ordered](i E, j E, etc ...E) E {
	return Top(func(x E, y E) bool { return y > x }, i, j, etc...)
}

// compare the two list, returns what elements are added and what elements are removed.
func Diff[E constraints.Ordered](newest []E, current []E) (added, removed []int) {
	x := 0
	y := 0

	added = make([]int, 0, len(newest))
	removed = make([]int, 0, len(current))

	sort.Slice(newest, func(i int, j int) bool { return newest[i] < newest[j] })
	sort.Slice(current, func(i int, j int) bool { return current[i] < current[j] })

	for {
		if x == len(newest) {
			removed = Reduce(current[y:], removed, func(r []int, i int, e E) []int {
				return append(r, y+i)
			})

			break
		}

		if y == len(current) {
			added = Reduce(newest[x:], added, func(r []int, i int, e E) []int {
				return append(r, x+i)
			})

			break
		}

		if newest[x] == current[y] {
			x++
			y++
		} else if newest[x] < current[y] {
			added = append(added, x)
			x++
		} else {
			removed = append(removed, y)
			y++
		}
	}

	return
}

// Convert slice element type to another type
func Slice[D any, O any](sli []O) (res []D) {
	dType := reflect.TypeOf(res)
	// oType := reflect.TypeOf(sli)

	dElemType := dType.Elem()
	// oElemType := oType.Elem()

	// if !oElemType.ConvertibleTo(dElemType) {
	// 	panic(fmt.Sprintf("invalide type convert, from %T to %T", sli, res))
	// }

	res = make([]D, 0, len(sli))
	for _, e := range sli {
		oValue := reflect.ValueOf(e)
		dValue := oValue.Convert(dElemType)
		res = append(res, dValue.Interface().(D))
	}

	return
}

func SafeSlice[T any](s *[]T) []T {
	if s == nil {
		return []T{}
	}

	return *s
}

func Reverse[T any](arr []T) []T {
	var reversed []T
	for i := len(arr) - 1; i >= 0; i-- {
		reversed = append(reversed, arr[i])
	}
	return reversed
}
