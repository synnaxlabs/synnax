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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/zyn"
)

const Version imex.Version = 0

// Data is the frozen type for log data at version 0. Channels are stored as bare
// integer keys. Key, Name, Type, and Version are envelope-level fields and are not part
// of Data.
type Data struct {
	Channels      []int `json:"channels" yaml:"channels" toml:"channels"`
	RemoteCreated bool  `json:"remote_created" yaml:"remote_created" toml:"remote_created"`
}

// Schema validates the wire shape of a v0 log payload.
var Schema = zyn.Object(map[string]zyn.Schema{
	"channels":       zyn.Array(zyn.Number().Int().Coerce()),
	"remote_created": zyn.Bool().Optional(),
})
