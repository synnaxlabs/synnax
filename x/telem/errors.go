package telem

import "github.com/cockroachdb/errors"

var (
	// InvalidRate is returned when a telem.Rate has an invalid value.
	InvalidRate = errors.New("[telem] - invalid rate")
	// InvalidDensity is returned when a telem.DataType has an invalid value.
	InvalidDensity = errors.New("[telem] - invalid data type")
	// InvalidTimeRange is returned when a telem.DataType has an invalid value.
	InvalidTimeRange = errors.New("[telem] - invalid time range")
	// InvalidDataType is returned when a telem.DataType has an invalid value.
	InvalidDataType = errors.New("[telem] - invalid data type")
)
