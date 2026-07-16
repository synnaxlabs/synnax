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
	"encoding/json"
	"fmt"

	"github.com/cespare/xxhash/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// hashConfig returns the xxhash64 of config as 16 lowercase hex characters, or an
// empty string when config cannot be encoded. encoding/json sorts map keys, so equal
// configs hash equally and an edit that is undone restores the original hash.
func hashConfig(config msgpack.EncodedJSON) string {
	if config == nil {
		config = msgpack.EncodedJSON{}
	}
	b, err := json.Marshal(map[string]any(config))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b))
}
