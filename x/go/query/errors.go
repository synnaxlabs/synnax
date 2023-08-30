// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package query

import "github.com/cockroachdb/errors"

var (
	Error = errors.New("[query] - error")
	// NotFound is returned when a requested entity cannot be found.
	NotFound = errors.Wrap(Error, "[query] - entity not found")
	// UniqueViolation is returned when a unique constraint on a particular entity
	// is violated.
	UniqueViolation = errors.Wrap(Error, "[query] - unique violation")
	// InvalidParameters is returned when a query has invalid parameters.
	InvalidParameters = errors.Wrap(Error, "[query] - invalid parameters")
)
