package algo

import "reflect"

func Count[T any](elements []T, f func(T) bool) int {
	result := 0
	for _, element := range elements {
		if f(element) {
			result++
		}
	}
	return result
}

func ContainSameElements[T comparable](lhs, rhs []T) bool {
	m1 := make(map[T]struct{}, len(lhs))
	for _, el := range lhs {
		m1[el] = struct{}{}
	}

	m2 := make(map[T]struct{}, len(rhs))
	for _, el := range rhs {
		m2[el] = struct{}{}
	}

	return reflect.DeepEqual(m1, m2)
}
