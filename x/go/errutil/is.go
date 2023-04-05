package errutil

import "github.com/cockroachdb/errors"

// IsAny returns true if err is any of the given errors.
func IsAny(err error, errs ...error) bool {
	for _, e := range errs {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
