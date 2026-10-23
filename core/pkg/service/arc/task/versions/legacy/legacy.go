// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy converts the Arc task config shapes released Cores and Consoles wrote:
// v0 {arcKey} plus the loop settings only a hand-written config carried, and v1 the
// typed shape whose loop mode was a raw execution_mode.
package legacy

import (
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// LastVersion is the newest legacy Arc task shape. The typed shape sits directly
// above it.
const LastVersion imex.Version = 1

// Config converts the released config shapes.
var Config = legacy.Rewrite{Post: func(config msgpack.EncodedJSON) {
	mode, ok := config["execution_mode"].(string)
	if !ok {
		return
	}
	delete(config, "execution_mode")
	config["performance"] = string(v4.PerformanceOf(v3.ExecutionMode(mode)))
}}
