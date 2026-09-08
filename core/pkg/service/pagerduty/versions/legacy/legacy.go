// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy converts the one PagerDuty config shape released Consoles wrote.
// Alerts carried an enabled flag; the typed shape stores disabled. The error severity
// flag was named for the mapping rather than the resulting state.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// LastVersion is the newest legacy PagerDuty shape. The typed shape sits directly
// above it.
const LastVersion imex.Version = 0

// Alert converts the released alert shape.
var Alert = legacy.Rewrite{Post: alert}

func alert(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "alerts", func(a msgpack.EncodedJSON) {
		legacy.FlipBool(a, "enabled", "disabled")
		legacy.RenameKey(a, "treat_error_as_critical", "errors_critical")
	})
}
