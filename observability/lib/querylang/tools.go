package querylang

func mapKeys[K comparable](in ...map[K]struct{}) []K {
	size := 0
	for i := range in {
		size += len(in[i])
	}
	result := make([]K, 0, size)

	for i := range in {
		for k := range in[i] {
			result = append(result, k)
		}
	}

	return result
}

func hasCondition(conditions []*Condition, hasF func(item *Condition) bool) bool {
	for i := range conditions {
		if hasF(conditions[i]) {
			return true
		}
	}

	return false
}

func filterConditions(conditions []*Condition, hasF func(item *Condition, index int) bool) []*Condition {
	result := make([]*Condition, 0, len(conditions))

	for i := range conditions {
		if hasF(conditions[i], i) {
			result = append(result, conditions[i])
		}
	}

	return result
}
