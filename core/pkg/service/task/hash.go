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
	"github.com/synnaxlabs/x/errors"
)

// hashConfig returns the xxhash64 of config as 16 lowercase hex characters. It returns
// an error when config cannot be JSON encoded, as would happen for a NaN or infinite
// float arriving over msgpack. encoding/json sorts map keys, so equal configs hash
// equally and an edit that is undone restores the original hash.
func hashConfig(config msgpack.EncodedJSON) (string, error) {
	if config == nil {
		config = msgpack.EncodedJSON{}
	}
	b, err := json.Marshal(map[string]any(config))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash task config")
	}
	return fmt.Sprintf("%016x", xxhash.Sum64(b)), nil
}

// configContent returns config without the record key, so hashes track config
// content rather than record identity: equal configs on different tasks hash
// equally.
func configContent(config msgpack.EncodedJSON) msgpack.EncodedJSON {
	content := make(msgpack.EncodedJSON, len(config))
	for k, v := range config {
		if k != "key" {
			content[k] = v
		}
	}
	return content
}
