package util

import "fmt"

func SafeExtract[T any](state *map[string]any, key string) (T, error) {
	var zero T
	if state == nil {
		return zero, fmt.Errorf("state is nil")
	}
	val, ok := (*state)[key]
	if !ok {
		return zero, fmt.Errorf("key %q not found in state", key)
	}
	typed, ok := val.(T)
	if !ok {
		return zero, fmt.Errorf("key %q has type %T, want %T", key, val, zero)
	}
	return typed, nil
}
