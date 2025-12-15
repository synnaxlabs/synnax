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
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type metric struct {
	ch      channel.Channel
	collect func() (float32, error)
}

func buildMetrics(s *storage.Layer) []metric {
	return []metric{
		{
			ch: channel.Channel{
				Name:     "mem_percentage",
				DataType: telem.Float32T,
			},
			collect: func() (float32, error) {
				vm, err := mem.VirtualMemory()
				if err != nil {
					return 0, err
				}
				return float32(vm.UsedPercent), err
			},
		},
		{
			ch: channel.Channel{
				Name:     "cpu_percentage",
				DataType: telem.Float32T,
			},
			collect: func() (float32, error) {
				cpuUsage, err := cpu.Percent(0, false)
				if err != nil {
					return 0, err
				}
				if len(cpuUsage) < 1 {
					return 0, errors.New("no cpu usage metric found")
				}
				return float32(cpuUsage[0]), err
			},
		},
		{
			ch: channel.Channel{
				Name:     "total_size_gb",
				DataType: telem.Float32T,
			},
			collect: func() (float32, error) {
				return float32(s.Size()) / float32(telem.Gigabyte), nil
			},
		},
		{
			ch: channel.Channel{
				Name:     "cesium_size_gb",
				DataType: telem.Float32T,
			},
			collect: func() (float32, error) {
				return float32(s.TSSize()) / float32(telem.Gigabyte), nil
			},
		},
		{
			ch: channel.Channel{
				Name:     "pebble_size_gb",
				DataType: telem.Float32T,
			},
			collect: func() (float32, error) {
				return float32(s.KVSize()) / float32(telem.Gigabyte), nil
			},
		},
	}
}
