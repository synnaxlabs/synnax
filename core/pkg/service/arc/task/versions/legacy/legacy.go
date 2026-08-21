// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy converts the Arc task config shape released Consoles stored and
// exported: {arcKey}, plus the loop settings only a hand-written config carried. The
// memory lock flag was named for the act rather than the resulting state.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// LastVersion is the newest legacy Arc task shape. The typed shape sits directly
// above it.
const LastVersion imex.Version = 0

// Config converts the released config shape.
var Config = legacy.Rewrite{Post: func(config msgpack.EncodedJSON) {
	legacy.RenameKey(config, "lock_memory", "memory_locked")
}}
