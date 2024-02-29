package algo

func SetsExceptUnion[K comparable](lhs, rhs map[K]struct{}) map[K]struct{} {
	result := make(map[K]struct{})

	for k, v := range lhs {
		if _, ok := rhs[k]; !ok {
			result[k] = v
		}
	}

	for k, v := range rhs {
		if _, ok := lhs[k]; !ok {
			result[k] = v
		}
	}

	return result
}
