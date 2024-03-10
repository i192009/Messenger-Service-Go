package xutil

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func GetKeys[T constraints.Ordered, V any](m map[T]V) []T {
	if len(m) == 0 {
		return nil
	}

	keys := make([]T, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func OrderForeach[M ~map[K]V, K constraints.Ordered, V any](m M, f func(k K, v V)) {
	keys := maps.Keys(m)
	slices.Sort(keys)
	for _, key := range keys {
		f(key, m[key])
	}
}

func Filter[M ~map[K]V, K constraints.Ordered, V any](m M, f func(k K, v V) bool) map[K]V {
	r := make(map[K]V)

	for k, v := range m {
		if f(k, v) {
			r[k] = v
		}
	}

	return r
}

func FilterKey[M ~map[K]V, K constraints.Ordered, V any](m M, f func(k K) bool) map[K]V {
	return Filter(m, func(k K, v V) bool { return f(k) })
}

func FilterVal[M ~map[K]V, K constraints.Ordered, V any](m M, f func(v V) bool) map[K]V {
	return Filter(m, func(k K, v V) bool { return f(v) })
}

func Merge[M ~map[K]V, K constraints.Ordered, V any](m1 M, m2 ...M) M {
	m := m1
	for _, e := range m2 {
		for k, v := range e {
			m[k] = v
		}
	}

	return m
}
