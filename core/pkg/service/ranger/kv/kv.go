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
	"strings"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// keySeparator splits the range from the key in the Gorp key of a Pair.
const keySeparator = "<--->"

// RangeOf returns the range of the pair whose Gorp key is key. It returns a validation
// error when key is not the key of a pair.
func RangeOf(key string) (ranger.Key, error) {
	rng, _, ok := strings.Cut(key, keySeparator)
	r, err := uuid.Parse(rng)
	if !ok || err != nil {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation,
			"%q is not the key of a range key-value pair",
			key,
		)
	}
	return r, nil
}
