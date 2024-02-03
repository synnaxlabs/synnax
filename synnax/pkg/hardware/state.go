/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package hardware

import (
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/hardware/task"
	"github.com/synnaxlabs/x/version"
)

type RackState struct {
	Heartbeat version.Heartbeat
	Modules   map[task.Key]ModuleState
}

type ModuleState struct {
	Running bool
}

type StateTracker struct {
	States map[rack.Key]RackState
}
