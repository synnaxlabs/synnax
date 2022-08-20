package validate

import (
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/types"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
)

type Validator struct {
	scope string
	errutil.Catch
}

func New(scope string) *Validator {
	return &Validator{scope: scope, Catch: *errutil.NewCatch(errutil.WithAggregation())}
}

func ternary(cond bool, err error) func() error {
	return func() error {
		return lo.Ternary(cond, err, nil)
	}
}

func NotNil(v *Validator, name string, value any) {
	v.Exec(ternary(
		value == nil,
		errors.Newf("[%s] - %s must be non-nil", v.scope, name)),
	)
}

func Positive[T types.Numeric](v *Validator, name string, value T) {
	v.Exec(ternary(
		value < 0,
		errors.Newf("[%s] - %s must be positive", v.scope, name)),
	)
}

func GreaterThan[T types.Numeric](v *Validator, name string, value T, threshold T) {
	v.Exec(ternary(
		value <= threshold,
		errors.Newf("[%s] - %s must be greater than %d", v.scope, name, threshold)),
	)
}

func GreaterThanEq[T types.Numeric](v *Validator, name string, value T, threshold T) {
	v.Exec(ternary(
		value < threshold,
		errors.Newf("[%s] - %s must be greater than or equal to %d", v.scope, name, threshold)),
	)
}

func NonZero[T types.Numeric](v *Validator, name string, value T) {
	v.Exec(ternary(
		value == 0,
		errors.Newf("[%s] - %s must be non-zero", v.scope, name)),
	)
}

func NotEmptySlice[T any](v *Validator, name string, value []T) {
	v.Exec(ternary(
		len(value) == 0,
		errors.Newf("[%s] - %s must be non-empty", v.scope, name)),
	)
}

func NotEmptyString[T ~string](v *Validator, name string, value T) {
	v.Exec(ternary(
		value == "",
		errors.Newf("[%s] - %s must be set", v.scope, name)),
	)
}
