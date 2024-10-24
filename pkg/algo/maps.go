package algo

import "fmt"

func MapMerge[T comparable](m1, m2 map[string]T) (map[string]T, error) {
	m := make(map[string]T)
	for k, v := range m1 {
		m[k] = v
	}
	for k, v := range m2 {
		if v1, ok := m[k]; ok {
			if v1 != v {
				return nil, fmt.Errorf("conflict with %s: \"%v\", \"%v\"", k, v, v1)
			}
		}
		m[k] = v
	}

	return m, nil
}
