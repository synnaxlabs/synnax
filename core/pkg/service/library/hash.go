// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"encoding/json/v2"
	"fmt"

	"github.com/cespare/xxhash/v2"
	"github.com/synnaxlabs/x/errors"
)

// Hash returns the xxhash64 of the library's entries as 16 lowercase hex characters.
// Equal entries hash equally; the library's name does not contribute.
func Hash(l Library) (string, error) {
	b, err := json.Marshal(l.Entries, json.Deterministic(true))
	if err != nil {
		return "", errors.Wrapf(err, "failed to hash library %s", l.Key)
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b)), nil
}
