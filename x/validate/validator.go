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
	return &Validator{scope: scope, Catch: *errutil.NewCatch(errutil.WithAggregation())}
}

func (v *Validator) Ternary(cond bool, msg string) {
	v.Exec(func() error {
		return lo.Ternary(cond, errors.Wrap(Error, msg), nil)
	})
}

func (v *Validator) Ternaryf(cond bool, format string, args ...any) {
	v.Exec(func() error {
		return lo.Ternary(cond,
			errors.Wrapf(Error, "[%s] - "+format, append([]any{v.scope}, args...)...),
			nil,
		)
	})
}

var (
	Error = errors.New("validation error")
)

func NotNil(v *Validator, name string, value any) {
	v.Ternaryf(value == nil, "%s must be non-nil", name)
}

func Positive[T types.Numeric](v *Validator, name string, value T) {
	v.Ternaryf(value <= 0, "%s must be positive", name)
}

func GreaterThan[T types.Numeric](v *Validator, name string, value T, threshold T) {
	v.Ternaryf(value <= threshold, "%s must be greater than %d", name, threshold)
}

func GreaterThanEq[T types.Numeric](v *Validator, name string, value T, threshold T) {
	v.Ternaryf(
		value < threshold,
		"%s must be greater than or equal to %d", v.scope, name, threshold)
}

func NonZero[T types.Numeric](v *Validator, name string, value T) {
	v.Ternaryf(
		value == 0,
		"%s must be non-zero", v.scope, name)
}

func NonNegative[T types.Numeric](v *Validator, name string, value T) {
	v.Ternaryf(
		value < 0,
		"%s must be non-negative", v.scope, name)
}

func NotEmptySlice[T any](v *Validator, name string, value []T) {
	v.Ternaryf(
		len(value) == 0,
		"%s must be non-empty", v.scope, name)
}

func NotEmptyString[T ~string](v *Validator, name string, value T) {
	v.Ternaryf(value == "", "%s must be set", v.scope, name)
}
