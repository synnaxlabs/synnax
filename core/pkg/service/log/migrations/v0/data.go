// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/x/errors"

const Version = 0

// Data is the frozen type for log data at version 0.0.0. Channels are stored as
// bare integer keys.
type Data struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Channels      []int  `json:"channels"`
	RemoteCreated bool   `json:"remote_created"`
}

// Validate enforces the structural invariants that the v0 Schema enforced
// implicitly: channels must be present (nil is treated as missing; empty slice
// is acceptable).
func (d Data) Validate() error {
	if d.Channels == nil {
		return errors.New("v0 log data: channels field is required")
	}
	return nil
}
