// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/x/zyn"

const Version = 0

// Data is the frozen type for log data at version 0.0.0. Channels are stored as
// bare integer keys.
type Data struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Channels      []int  `json:"channels"`
	RemoteCreated bool   `json:"remote_created"`
}

// Schema validates the full log resource at version 0.0.0.
var Schema = zyn.Object(map[string]zyn.Schema{
	"key":            zyn.String().Optional(),
	"name":           zyn.String().Optional(),
	"channels":       zyn.Array(zyn.Number()),
	"remote_created": zyn.Bool(),
})
