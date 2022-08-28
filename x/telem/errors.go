package telem

import "github.com/cockroachdb/errors"

var (
	InvalidRate      = errors.New("[telem] - invalid data rate")
	InvalidDensity   = errors.New("[telem] - invalid data type")
	InvalidTimeRange = errors.New("[telem] - invalid time range")
	InvalidDataType  = errors.New("[telem] - invalid data type")
)
