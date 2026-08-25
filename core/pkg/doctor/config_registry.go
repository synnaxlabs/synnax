// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"github.com/google/uuid"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/labjack"
	"github.com/synnaxlabs/synnax/pkg/service/modbus"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	"github.com/synnaxlabs/synnax/pkg/service/opcua"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	racktask "github.com/synnaxlabs/synnax/pkg/service/rack/task"
	"github.com/synnaxlabs/x/gorp"
)

// newConfigRegistry builds the registry of task configuration tables. Each hardware
// integration stores its task configurations in its own table, keyed by task key.
func newConfigRegistry() []table {
	return []table{
		newConfigTable[ni.AnalogReadConfig](),
		newConfigTable[ni.AnalogWriteConfig](),
		newConfigTable[ni.CounterReadConfig](),
		newConfigTable[ni.DigitalReadConfig](),
		newConfigTable[ni.DigitalWriteConfig](),
		newConfigTable[ni.ScanConfig](),
		newConfigTable[opcua.ReadConfig](),
		newConfigTable[opcua.WriteConfig](),
		newConfigTable[opcua.ScanConfig](),
		newConfigTable[labjack.ReadConfig](),
		newConfigTable[labjack.WriteConfig](),
		newConfigTable[labjack.ScanConfig](),
		newConfigTable[modbus.ReadConfig](),
		newConfigTable[modbus.WriteConfig](),
		newConfigTable[modbus.ScanConfig](),
		newConfigTable[ethercat.ReadConfig](),
		newConfigTable[ethercat.WriteConfig](),
		newConfigTable[ethercat.ScanConfig](),
		newConfigTable[http.ReadConfig](),
		newConfigTable[http.WriteConfig](),
		newConfigTable[http.ScanConfig](),
		newConfigTable[pagerduty.TaskConfig](),
		newConfigTable[racktask.StatusConfig](),
		newConfigTable[arctask.Config](),
	}
}

// newConfigTable builds the table for one task configuration record type. Records are
// keyed by the key of the task they configure.
func newConfigTable[E gorp.Entry[uuid.UUID]]() table {
	return newTable(tableConfig[uuid.UUID, E]{
		collect: func(s *state, e E) { s.configs.Add(e.GorpKey()) },
		check: func(s *state, e E) {
			if key := e.GorpKey(); !s.tasks.Contains(key) {
				s.note(CheckTaskConfig, "config record has no task", key.String())
			}
		},
	})
}
