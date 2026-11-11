// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package config

import (
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ValidateStreamRate adds an error to v if the stream rate of a read task is not
// positive or is greater than its sample rate.
func ValidateStreamRate(v *validate.Validator, sampleRate, streamRate telem.Rate) {
	v.GreaterThan("stream_rate", streamRate, 0)
	v.Ternary(
		"stream_rate",
		streamRate > sampleRate,
		"must be less than or equal to the sample rate",
	)
}
