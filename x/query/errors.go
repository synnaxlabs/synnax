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
	// NotFound is returned when a requested entity cannot be found.
	NotFound = errors.New("[query] - entity not found")
	// UniqueViolation is returned when a unique constraint on a particular entity
	// is violated.
	UniqueViolation = errors.New("[query] - unique violation")
)
