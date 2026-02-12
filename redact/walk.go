package redact

const maxWalkDepth = 16

// walkAny applies fn to every string leaf in v, recursively.
func walkAny(v any, fn func(string) string) any {
	return walkDepth(v, fn, 0)
}

func walkDepth(v any, fn func(string) string, depth int) any {
	if depth > maxWalkDepth {
		return v
	}
	switch val := v.(type) {
	case string:
		return fn(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, child := range val {
			out[k] = walkDepth(child, fn, depth+1)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, child := range val {
			out[i] = walkDepth(child, fn, depth+1)
		}
		return out
	default:
		return v
	}
}
