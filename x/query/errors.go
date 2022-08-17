package query

import "github.com/cockroachdb/errors"

var (
	// NotFound is returned when a requested entity cannot be found.
	NotFound = errors.New("[query] - entity not found")
	// UniqueViolation is returned when a unique constraint on a particular entity
	// is violated.
	UniqueViolation = errors.New("[query] - unique violation")
)
