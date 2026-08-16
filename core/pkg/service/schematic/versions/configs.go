// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// NormalizeConfigKeys rewrites raw's camelCase keys, recursively, into the snake_case
// the union decodes. It returns nil when raw is nil or holds no object.
func NormalizeConfigKeys(raw msgpack.EncodedJSON) msgpack.EncodedJSON {
	return v7.NormalizeConfigKeys(raw)
}

// DecodeElementConfig decodes normalized fields into the element config union. It
// errors when raw carries no variant or one the union does not name.
func DecodeElementConfig(raw msgpack.EncodedJSON) (ElementConfig, error) {
	return v7.DecodeElementConfig(raw)
}

// ElementConfigFields re-encodes cfg into its normalized field map, inverting
// DecodeElementConfig.
func ElementConfigFields(cfg ElementConfig) (msgpack.EncodedJSON, error) {
	return v7.ElementConfigFields(cfg)
}
