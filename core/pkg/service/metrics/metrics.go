// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics

import (
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type metric struct {
	ch      channel.Channel
	collect func() (float32, error)
}

var all = []metric{
	{
		ch: channel.Channel{Name: "mem_percentage", DataType: telem.Float32T},
		collect: func() (float32, error) {
			vm, err := mem.VirtualMemory()
			if err != nil {
				return 0, err
			}
			return float32(vm.UsedPercent), nil
		},
	},
	{
		ch: channel.Channel{Name: "cpu_percentage", DataType: telem.Float32T},
		collect: func() (float32, error) {
			cpuUsage, err := cpu.Percent(0, false)
			if err != nil {
				return 0, err
			}
			if len(cpuUsage) < 1 {
				return 0, errors.New("no cpu usage metric found")
			}
			return float32(cpuUsage[0]), nil
		},
	},
}
