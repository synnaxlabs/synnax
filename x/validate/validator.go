package validate

import (
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/types"
)

type Validator struct {
	scope string
	errutil.Catch
}

func New(scope string) *Validator {
	return &Validator{scope: scope, Catch: *errutil.NewCatch()}
}

func (v *Validator) Ternary(cond bool, msg string) bool {
	v.Exec(func() error {
		return lo.Ternary(cond, v.New(msg), nil)
	})
	return v.Error() != nil
}

func (v *Validator) Ternaryf(cond bool, format string, args ...any) bool {
	v.Exec(func() error {
		return lo.Ternary(cond, v.Newf(format, args...), nil)
	})
	return v.Error() != nil
}

func (v *Validator) New(msg string) error {
	return errors.Wrapf(Error, "[%s] - "+msg, v.scope)
}

func (v *Validator) Newf(format string, args ...any) error {
	return errors.Wrapf(Error, "[%s] - "+format, append([]any{v.scope}, args...)...)
}

func (v *Validator) Funcf(f func() bool, format string, args ...any) bool {
	v.Exec(func() error {
		return lo.Ternary(f(), v.Newf(format, args...), nil)
	})
	return v.Error() != nil
}

func (v *Validator) Func(f func() bool, msg string) bool {
	v.Exec(func() error {
		return lo.Ternary(f(), v.New(msg), nil)
	})
	return v.Error() != nil
}

var (
	Error = errors.New("validation error")
)

func NotNil(v *Validator, name string, value any) bool {
	return v.Ternaryf(value == nil, "%s must be non-nil", name)
}

func Positive[T types.Numeric](v *Validator, name string, value T) bool {
	return v.Ternaryf(value <= 0, "%s must be positive", name)
}

func GreaterThan[T types.Numeric](v *Validator, name string, value T, threshold T) bool {
	return v.Ternaryf(value <= threshold, "%s must be greater than %d", name, threshold)
}

func GreaterThanEq[T types.Numeric](v *Validator, name string, value T, threshold T) bool {
	return v.Ternaryf(
		value < threshold,
		"%s must be greater than or equal to %d", name, threshold)
}

func NonZero[T types.Numeric](v *Validator, name string, value T) bool {
	return v.Ternaryf(
		value == 0,
		"%s must be non-zero", name)
}

func NonNegative[T types.Numeric](v *Validator, name string, value T) bool {
	return v.Ternaryf(
		value < 0,
		"%s must be non-negative", name)
}

func NotEmptySlice[T any](v *Validator, name string, value []T) bool {
	return v.Ternaryf(
		len(value) == 0,
		"%s must be non-empty", name)
}

func NotEmptyString[T ~string](v *Validator, name string, value T) bool {
	return v.Ternaryf(value == "", "%s must be set", name)
}

func MapDoesNotContainF[K comparable, V any](
	v *Validator,
	value K,
	m map[K]V,
	format string,
	args ...any,
) bool {
	return v.Funcf(func() bool {
		_, ok := m[value]
		return ok
	}, format, args...)
}

func MapContainsf[K comparable, V any](
	v *Validator,
	value K,
	m map[K]V,
	format string,
	args ...any,
) bool {
	return v.Funcf(func() bool {
		_, ok := m[value]
		return !ok
	}, format, args...)
}
