package telem

import "github.com/cockroachdb/errors"

var (
	InvalidDataRate  = errors.New("[telem] - invalid data rate")
	InvalidDataType  = errors.New("[telem] - invalid data type")
	InvalidTimeRange = errors.New("[telem] - invalid time range")
)
