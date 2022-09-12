package override

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/types"
	"reflect"
)

func Numeric[N types.Numeric](value N, override N) N {
	return lo.Ternary(override != 0, override, value)
}

func String[T ~string](value T, override T) T {
	return lo.Ternary(override != "", override, value)
}

func Nil[T any](value T, override T) T {
	v := reflect.ValueOf(override)
	if !v.IsValid() || v.IsZero() || v.IsNil() {
		return value
	}
	return override
}

func Slice[T any](value []T, override []T) []T {
	return lo.Ternary(len(override) > 0, override, value)
}
