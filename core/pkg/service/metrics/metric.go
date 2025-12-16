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
	ch channel.Channel
	// Go does not allow unions, so we use an any type here.
	// Data types for the channels are float32 and int32.
	collect func() (any, error)
}

func buildMetrics(s *storage.Layer, channelCount func() int) []metric {
	return []metric{
		{
			ch: channel.Channel{
				Name:     "mem_percentage",
				DataType: telem.Float32T,
			},
			collect: func() (any, error) {
				vm, err := mem.VirtualMemory()
				if err != nil {
					return float32(0), err
				}
				return float32(vm.UsedPercent), err
			},
		},
		{
			ch: channel.Channel{
				Name:     "cpu_percentage",
				DataType: telem.Float32T,
			},
			collect: func() (any, error) {
				cpuUsage, err := cpu.Percent(0, false)
				if err != nil {
					return float32(0), err
				}
				if len(cpuUsage) < 1 {
					return float32(0), errors.New("no cpu usage metric found")
				}
				return float32(cpuUsage[0]), err
			},
		},
		{
			ch: channel.Channel{
				Name:     "total_size_gb",
				DataType: telem.Float32T,
			},
			collect: func() (any, error) {
				return float32(s.Size()) / float32(telem.Gigabyte), nil
			},
		},
		{
			ch: channel.Channel{
				Name:     "ts_size_gb",
				DataType: telem.Float32T,
			},
			collect: func() (any, error) {
				return float32(s.TSSize()) / float32(telem.Gigabyte), nil
			},
		},
		{
			ch: channel.Channel{
				Name:     "kv_size_gb",
				DataType: telem.Float32T,
			},
			collect: func() (any, error) {
				return float32(s.KVSize()) / float32(telem.Gigabyte), nil
			},
		},
		{
			ch: channel.Channel{
				Name:     "channel_count",
				DataType: telem.Int32T,
			},
			collect: func() (any, error) {
				return int32(channelCount()), nil
			},
		},
	}
}
