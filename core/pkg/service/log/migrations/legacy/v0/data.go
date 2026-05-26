// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
)

const Version imex.Version = 0

// Data is the frozen type for log data at version 0. Channels are stored as bare channel
// keys. Key, Name, Type, and Version are envelope-level fields and are not part of Data.
type Data struct {
	Channels      []channel.Key `json:"channels"`
	RemoteCreated bool          `json:"remoteCreated"`
}

// Parse marshals the raw map back to JSON and unmarshals it into a typed Data. Channel
// keys flow directly into the typed []channel.Key; a fractional or out-of-range value,
// or a non-array channels field, fails json.Unmarshal.
func Parse(raw map[string]any) (Data, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return Data{}, err
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return Data{}, err
	}
	return d, nil
}
