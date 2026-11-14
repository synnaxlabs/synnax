// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package exit maps the error that ends the synnax command to its exit code.
package exit

import (
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

const (
	// Failure is the code of an error that a later start can get past.
	Failure = 1
	// DataRefused is the code of a Core that refuses its data until a person acts. It
	// is the BSD code for a data format error.
	DataRefused = 65
	// DataLocked is the code of a Core whose data directory another process holds. It
	// is the BSD code for a temporary failure: a start after that process exits works.
	DataLocked = 75
	// Config is the code of a flag or config value that a person must fix. It is the
	// BSD code for a configuration error.
	Config = 78
)

// Code returns the exit code for err, or 0 when err is nil.
func Code(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, storage.ErrRefused):
		return DataRefused
	case errors.Is(err, storage.ErrLocked):
		return DataLocked
	case errors.Is(err, validate.ErrValidation):
		return Config
	default:
		return Failure
	}
}
