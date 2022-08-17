package address

import "github.com/cockroachdb/errors"

var NotFound = errors.New("[address] - not found")

func TargetNotFound(target Address) error {
	return errors.Wrapf(NotFound, "[address] - target %s not found", target)
}
