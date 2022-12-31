// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
