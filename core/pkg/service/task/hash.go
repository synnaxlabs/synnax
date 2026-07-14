// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"fmt"

	"github.com/cespare/xxhash/v2"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// HashConfig returns the xxhash64 of the canonical JSON form of config as a
// 16-character lowercase hex string. The C++ and TypeScript clients produce the
// same hash for the same config; StatusDetails.ConfigHash carries it to detect
// config drift. Returns an empty string when config cannot be canonicalized.
func HashConfig(config msgpack.EncodedJSON) string {
	b, err := xjson.Canonical(map[string]any(config))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b))
}

// NewStatusDetails builds StatusDetails for a status reported by the live
// instance of t, stamping the hash of the config the instance was built from
// and the rack it runs on.
func NewStatusDetails(t Task, running bool) StatusDetails {
	return StatusDetails{
		Task:       t.Key,
		Running:    running,
		ConfigHash: HashConfig(t.Config),
		Rack:       t.Rack,
	}
}
