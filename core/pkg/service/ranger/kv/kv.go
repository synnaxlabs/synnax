// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// RangeOf returns the range of the pair whose Gorp key is key. It returns a validation
// error when the range part of key is not a UUID.
func RangeOf(key string) (ranger.Key, error) {
	r, err := uuid.Parse(Pair{}.GorpKeyRange(key))
	if err != nil {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation,
			"%q is not the key of a range key-value pair",
			key,
		)
	}
	return r, nil
}
