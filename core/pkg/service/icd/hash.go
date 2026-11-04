// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"encoding/json/v2"
	"fmt"

	"github.com/cespare/xxhash/v2"
	"github.com/synnaxlabs/x/errors"
)

// Hash returns the hash of the ICD's messages as 16 lowercase hex characters. The Core
// stores it in the ICD's Hash field, and a writer sends the hash it read so the Core
// can reject a stale save. A rename leaves it unchanged.
func Hash(ic ICD) (string, error) {
	b, err := json.Marshal(ic.Messages, json.Deterministic(true))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash the ICD's messages")
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b)), nil
}
