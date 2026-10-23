// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package library is the toolkit that domain services use to store library documents:
// resources such as an ICD that hold definitions many other resources use. It hashes
// document content, collects the problems a check finds, rejects saves of a stale
// document, and records which resources use a document. The toolkit never calls back
// into a domain; each domain calls the parts it needs.
package library

import (
	"encoding/json/v2"
	"fmt"

	"github.com/cespare/xxhash/v2"
	"github.com/synnaxlabs/x/errors"
)

// Hash returns the xxhash64 of v's deterministic JSON as 16 lowercase hex characters.
// Equal values hash equally.
func Hash(v any) (string, error) {
	b, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash library content")
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b)), nil
}
