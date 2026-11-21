// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

const (
	tsSizeMetricName = "ts_size_gb"
	kvSizeMetricName = "kv_size_gb"
)

type metric struct {
	collect func() (float32, error)
	ch      channel.Channel
}

func (s *Service) createMetrics(namePrefix string, idxKey channel.LocalKey) []metric {
	makeChannel := func(name string) channel.Channel {
		return channel.Channel{
			Name:       namePrefix + name,
			LocalIndex: idxKey,
			DataType:   telem.Float32T,
		}
	}
	metrics := []metric{
		{
			ch: makeChannel("mem_percentage"),
			collect: func() (float32, error) {
				vm, err := mem.VirtualMemory()
				if err != nil {
					return 0, err
				}
				return float32(vm.UsedPercent), nil
			},
		},
		{
			ch: makeChannel("cpu_percentage"),
			collect: func() (float32, error) {
				cpuUsage, err := cpu.Percent(0, false)
				if err != nil {
					return 0, err
				}
				if len(cpuUsage) < 1 {
					return 0, errors.New("no CPU usage metric found")
				}
				return float32(cpuUsage[0]), nil
			},
		},
		{
			ch: makeChannel(tsSizeMetricName),
			collect: func() (float32, error) {
				return float32(s.cfg.Storage.TSSize().Gigabytes()), nil
			},
		},
		{
			ch: makeChannel(kvSizeMetricName),
			collect: func() (float32, error) {
				return float32(s.cfg.Storage.KVSize().Gigabytes()), nil
			},
		},
	}
	return metrics
}

// createCalculatedMetrics returns the calculated metrics whose inputs opened, given
// each opened metric by the name that the service gives it. A metric reads each input
// by its current name, quoted when a user renamed it.
func createCalculatedMetrics(
	namePrefix string,
	opened map[string]channel.Channel,
) []channel.Channel {
	tsName, kvName := namePrefix+tsSizeMetricName, namePrefix+kvSizeMetricName
	ts, tsOpened := opened[tsName]
	kv, kvOpened := opened[kvName]
	if !tsOpened || !kvOpened {
		return nil
	}
	return []channel.Channel{{
		Name:     namePrefix + "total_size_gb",
		DataType: telem.Float32T,
		Expression: fmt.Sprintf(
			"return %s + %s",
			inputText(ts.Name, tsName),
			inputText(kv.Name, kvName),
		),
	}}
}

// inputText returns the text that reads the input named name, whose given name is
// given: bare while it keeps that name, and quoted otherwise.
func inputText(name, given string) string {
	if name == given {
		return name
	}
	return channel.QuotePath(name)
}
